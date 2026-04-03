using Microsoft.AspNetCore.SignalR;
using PKHeX.Core;
using SysBot.Base;
using SysBot.Pokemon;
using SysBot.Pokemon.Web;
using SysBot.Pokemon.Web.Realtime;
using SysBot.Pokemon.Z3;
using System.Text.Json;
using System.Text.Json.Serialization;

// --- Configuration ---

const string ConfigPath = "config.json";

// Load or create config.json using the same pattern as ConsoleApp
ProgramConfig cfg;
if (!File.Exists(ConfigPath))
{
    // Seed a default config so the user has something to edit
    var bot = new PokeBotState
    {
        Connection = new SwitchConnectionConfig { IP = "192.168.0.1", Port = 6000 },
        InitialRoutine = PokeRoutineType.FlexTrade,
    };
    cfg = new ProgramConfig { Bots = [bot] };
    var json = JsonSerializer.Serialize(cfg, ProgramConfigContext.Default.ProgramConfig);
    File.WriteAllText(ConfigPath, json);
    Console.WriteLine("Created default config.json — configure it and restart.");
}
else
{
    var json = File.ReadAllText(ConfigPath);
    cfg = JsonSerializer.Deserialize(json, ProgramConfigContext.Default.ProgramConfig) ?? new ProgramConfig();
}

// --- Bot runner ---

// Create the appropriate runner for the configured game mode
var runner = GetRunner(cfg);

// Initialize the Z3 seed checker (Sword/Shield specific, but always wired up)
PokeTradeBotSWSH.SeedChecker = new Z3SeedSearchHandler<PK8>();

// Add bots from config
foreach (var bot in cfg.Bots)
{
    bot.Initialize();
    if (!AddBot(runner, bot, cfg.Mode))
        Console.WriteLine($"Failed to add bot: {bot}");
}

// --- ASP.NET Core host ---

var urls = Environment.GetEnvironmentVariable("ASPNETCORE_URLS") ?? "http://0.0.0.0:5050";

var builder = WebApplication.CreateBuilder(args);
builder.WebHost.UseUrls(urls);

// Register the bot runner and config as singletons so controllers can inject them
builder.Services.AddSingleton<IPokeBotRunner>(runner);
builder.Services.AddSingleton(cfg);

// SignalR for real-time log streaming
builder.Services.AddSignalR();

// API controllers with System.Text.Json defaults
builder.Services.AddControllers()
    .AddJsonOptions(opts =>
    {
        opts.JsonSerializerOptions.PropertyNamingPolicy = JsonNamingPolicy.CamelCase;
        opts.JsonSerializerOptions.WriteIndented = false;
        // Serialize enums as their string names for clean API responses
        opts.JsonSerializerOptions.Converters.Add(new JsonStringEnumConverter());
    });

var app = builder.Build();

// --- Middleware pipeline ---

// Serve the React SPA from wwwroot (copied from web/dist at build time)
app.UseDefaultFiles();
app.UseStaticFiles();

// Map API controllers and the SignalR hub
app.MapControllers();
app.MapHub<LogHub>("/hubs/logs");

// SPA fallback — any unmatched route serves index.html so client-side routing works
app.MapFallbackToFile("index.html");

// --- Logging wiring ---

// Forward LogUtil messages to SignalR clients
var hubContext = app.Services.GetRequiredService<IHubContext<LogHub>>();
LogUtil.Forwarders.Add(new WebLogForwarder(hubContext));

// Also keep console output for debugging
LogUtil.Forwarders.Add(ConsoleForwarder.Instance);

// Forward echo messages to SignalR clients
EchoUtil.Forwarders.Add(message =>
{
    var timestamp = DateTime.UtcNow.ToString("o");
    _ = hubContext.Clients.All.SendAsync(LogHub.ReceiveEcho, timestamp, message);
});

// --- Start bots and web host ---

runner.StartAll();
Console.WriteLine($"Started all bots (Count: {cfg.Bots.Length}).");

// Graceful shutdown: stop all bots when the host is stopping
app.Lifetime.ApplicationStopping.Register(() =>
{
    Console.WriteLine("Application stopping — shutting down all bots...");
    runner.StopAll();
});

Console.WriteLine($"Web host listening on {urls}");
app.Run();

// --- Helper methods ---

static IPokeBotRunner GetRunner(ProgramConfig prog) => prog.Mode switch
{
    ProgramMode.SWSH => new PokeBotRunnerImpl<PK8>(prog.Hub, new BotFactory8SWSH()),
    ProgramMode.BDSP => new PokeBotRunnerImpl<PB8>(prog.Hub, new BotFactory8BS()),
    ProgramMode.LA   => new PokeBotRunnerImpl<PA8>(prog.Hub, new BotFactory8LA()),
    ProgramMode.SV   => new PokeBotRunnerImpl<PK9>(prog.Hub, new BotFactory9SV()),
    ProgramMode.LZA  => new PokeBotRunnerImpl<PA9>(prog.Hub, new BotFactory9LZA()),
    _ => throw new IndexOutOfRangeException("Unsupported mode."),
};

static bool AddBot(IPokeBotRunner env, PokeBotState botCfg, ProgramMode mode)
{
    if (!botCfg.IsValid())
    {
        Console.WriteLine($"{botCfg}'s config is not valid.");
        return false;
    }

    PokeRoutineExecutorBase newBot;
    try
    {
        newBot = env.CreateBotFromConfig(botCfg);
    }
    catch
    {
        Console.WriteLine($"Current Mode ({mode}) does not support this type of bot ({botCfg.CurrentRoutineType}).");
        return false;
    }

    try
    {
        env.Add(newBot);
    }
    catch (ArgumentException ex)
    {
        Console.WriteLine(ex.Message);
        return false;
    }

    Console.WriteLine($"Added: {botCfg}: {botCfg.InitialRoutine}");
    return true;
}
