using PKHeX.Core;
using SysBot.Pokemon.Discord;
using SysBot.Pokemon.Twitch;
using System.Threading;
using System.Threading.Tasks;

namespace SysBot.Pokemon.Web;

/// <summary>
/// Bot Environment implementation with Discord and Twitch integrations.
/// Mirrors the ConsoleApp version but lives in the Web project's namespace.
/// </summary>
public class PokeBotRunnerImpl<T> : PokeBotRunner<T> where T : PKM, new()
{
    public PokeBotRunnerImpl(PokeTradeHub<T> hub, BotFactory<T> fac) : base(hub, fac) { }
    public PokeBotRunnerImpl(PokeTradeHubConfig config, BotFactory<T> fac) : base(config, fac) { }

    private TwitchBot<T>? Twitch;

    protected override void AddIntegrations()
    {
        AddDiscordBot(Hub.Config.Discord);
        AddTwitchBot(Hub.Config.Twitch);
    }

    private void AddTwitchBot(TwitchSettings config)
    {
        if (string.IsNullOrWhiteSpace(config.Token))
            return;
        if (Twitch != null)
            return; // already created

        if (string.IsNullOrWhiteSpace(config.Channel))
            return;
        if (string.IsNullOrWhiteSpace(config.Username))
            return;

        Twitch = new TwitchBot<T>(config, Hub);
        if (config.DistributionCountDown)
            Hub.BotSync.BarrierReleasingActions.Add(() => Twitch.StartingDistribution(config.MessageStart));
    }

    private void AddDiscordBot(DiscordSettings config)
    {
        var token = config.Token;
        if (string.IsNullOrWhiteSpace(token))
            return;

        var bot = new SysCord<T>(this);
        Task.Run(() => bot.MainAsync(token, CancellationToken.None), CancellationToken.None);
    }
}
