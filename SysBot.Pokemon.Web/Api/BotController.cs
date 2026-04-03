using System;
using System.Linq;
using Microsoft.AspNetCore.Mvc;
using SysBot.Base;
using SysBot.Pokemon.Web.Models;

namespace SysBot.Pokemon.Web.Api;

/// <summary>
/// Manages bot instances: list, add, remove, and control lifecycle (start/stop/pause/resume/restart).
/// </summary>
[ApiController]
[Route("api/bots")]
public class BotController(IPokeBotRunner runner) : ControllerBase
{
    // Cast once — the runner is always a BotRunner<PokeBotState> at runtime.
    private BotRunner<PokeBotState> Runner => (BotRunner<PokeBotState>)runner;

    /// <summary>GET /api/bots — list every registered bot.</summary>
    [HttpGet]
    public IActionResult List()
    {
        var dtos = Runner.Bots.Select(MapToDto).ToList();
        return Ok(dtos);
    }

    /// <summary>POST /api/bots — register a new bot from connection details.</summary>
    [HttpPost]
    public IActionResult Add([FromBody] AddBotRequest request)
    {
        // Parse the protocol enum.
        if (!Enum.TryParse<SwitchProtocol>(request.Protocol, ignoreCase: true, out var protocol))
            return BadRequest(new { error = $"Invalid protocol '{request.Protocol}'. Expected: {string.Join(", ", Enum.GetNames<SwitchProtocol>())}" });

        // Parse the routine enum.
        if (!Enum.TryParse<PokeRoutineType>(request.Routine, ignoreCase: true, out var routine))
            return BadRequest(new { error = $"Invalid routine '{request.Routine}'. Expected: {string.Join(", ", Enum.GetNames<PokeRoutineType>())}" });

        // Verify the current runner supports this routine type.
        if (!runner.SupportsRoutine(routine))
            return BadRequest(new { error = $"Routine '{routine}' is not supported by the current game mode." });

        // Build the connection config and bot state.
        var config = new SwitchConnectionConfig
        {
            IP = request.Ip,
            Port = request.Port,
            Protocol = protocol,
        };
        var state = new PokeBotState { Connection = config };
        state.Initialize(routine);

        // Check for duplicates (BotRunner.Add throws if the connection already exists).
        var existing = Runner.Bots.Find(b => b.Bot.Connection.Name == config.ToString());
        if (existing is not null)
            return Conflict(new { error = $"A bot with connection '{config}' is already registered." });

        // Create the executor and register it.
        var bot = runner.CreateBotFromConfig(state);
        runner.Add(bot);

        // Find the newly-added source so we can return a full DTO.
        var source = runner.GetBot(state);
        var dto = source is not null ? MapToDto(source) : null;

        return CreatedAtAction(nameof(List), dto);
    }

    /// <summary>DELETE /api/bots/{id} — remove a bot by its connection name.</summary>
    [HttpDelete("{id}")]
    public IActionResult Remove(string id)
    {
        var source = FindBot(id);
        if (source is null)
            return NotFound(new { error = $"Bot '{id}' not found." });

        runner.Remove(source.Bot.Config, callStop: true);
        return NoContent();
    }

    // ── Lifecycle actions ───────────────────────────────────────────────

    /// <summary>POST /api/bots/{id}/start — start a single bot.</summary>
    [HttpPost("{id}/start")]
    public IActionResult Start(string id)
    {
        var source = FindBot(id);
        if (source is null)
            return NotFound(new { error = $"Bot '{id}' not found." });

        // Ensure one-time initialization has happened.
        if (!runner.RunOnce)
            runner.InitializeStart();

        source.Start();
        return Ok(MapToDto(source));
    }

    /// <summary>POST /api/bots/{id}/stop — stop a single bot.</summary>
    [HttpPost("{id}/stop")]
    public IActionResult Stop(string id)
    {
        var source = FindBot(id);
        if (source is null)
            return NotFound(new { error = $"Bot '{id}' not found." });

        source.Stop();
        return Ok(MapToDto(source));
    }

    /// <summary>POST /api/bots/{id}/pause — pause a single bot.</summary>
    [HttpPost("{id}/pause")]
    public IActionResult Pause(string id)
    {
        var source = FindBot(id);
        if (source is null)
            return NotFound(new { error = $"Bot '{id}' not found." });

        source.Pause();
        return Ok(MapToDto(source));
    }

    /// <summary>POST /api/bots/{id}/resume — resume a paused bot.</summary>
    [HttpPost("{id}/resume")]
    public IActionResult Resume(string id)
    {
        var source = FindBot(id);
        if (source is null)
            return NotFound(new { error = $"Bot '{id}' not found." });

        source.Resume();
        return Ok(MapToDto(source));
    }

    /// <summary>POST /api/bots/{id}/restart — restart a single bot (reset connection then start).</summary>
    [HttpPost("{id}/restart")]
    public IActionResult Restart(string id)
    {
        var source = FindBot(id);
        if (source is null)
            return NotFound(new { error = $"Bot '{id}' not found." });

        source.Restart();
        return Ok(MapToDto(source));
    }

    // ── Bulk lifecycle ──────────────────────────────────────────────────

    /// <summary>POST /api/bots/start-all — start every registered bot.</summary>
    [HttpPost("start-all")]
    public IActionResult StartAll()
    {
        runner.StartAll();
        return Ok(new { started = Runner.Bots.Count });
    }

    /// <summary>POST /api/bots/stop-all — stop every registered bot.</summary>
    [HttpPost("stop-all")]
    public IActionResult StopAll()
    {
        runner.StopAll();
        return Ok(new { stopped = Runner.Bots.Count });
    }

    // ── Helpers ─────────────────────────────────────────────────────────

    /// <summary>Find a <see cref="BotSource{T}"/> by its connection name (IP or USB port).</summary>
    private BotSource<PokeBotState>? FindBot(string id) =>
        Runner.Bots.Find(b => b.Bot.Connection.Name == id);

    /// <summary>Map a <see cref="BotSource{T}"/> to a serializable <see cref="BotDto"/>.</summary>
    private static BotDto MapToDto(BotSource<PokeBotState> src)
    {
        var bot = src.Bot;
        var cfg = bot.Config;
        var conn = cfg.Connection;

        return new BotDto
        {
            Id = bot.Connection.Name,
            Ip = conn.IP,
            Port = conn.Port,
            Protocol = conn.Protocol.ToString(),
            InitialRoutine = cfg.InitialRoutine.ToString(),
            CurrentRoutine = cfg.CurrentRoutineType.ToString(),
            NextRoutine = cfg.NextRoutineType.ToString(),
            IsRunning = src.IsRunning,
            IsPaused = src.IsPaused,
            IsConnected = bot.Connection.Connected,
            LastLog = bot.LastLogged,
            LastActive = bot.LastTime,
        };
    }
}
