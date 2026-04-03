using System;
using System.Linq;
using System.Collections.Generic;
using Microsoft.AspNetCore.Mvc;
using SysBot.Base;

namespace SysBot.Pokemon.Web.Api;

/// <summary>
/// Lightweight status endpoints: program metadata and queue counts.
/// </summary>
[ApiController]
[Route("api")]
public class StatusController(IPokeBotRunner runner, ProgramConfig programConfig) : ControllerBase
{
    /// <summary>
    /// GET /api/meta — high-level metadata about the running instance:
    /// game mode, supported routines, available protocols, and running state.
    /// </summary>
    [HttpGet("meta")]
    public IActionResult GetMeta()
    {
        // Collect every routine the current game-mode factory can handle.
        var supportedRoutines = Enum.GetValues<PokeRoutineType>()
            .Where(runner.SupportsRoutine)
            .Select(r => r.ToString())
            .ToList();

        // All protocol options (WiFi, USB).
        var protocols = Enum.GetNames<SwitchProtocol>().ToList();

        return Ok(new
        {
            mode = programConfig.Mode.ToString(),
            supportedRoutines,
            protocols,
            isRunning = runner.IsRunning,
        });
    }

    /// <summary>
    /// GET /api/queues — current queue depths for each trade routine,
    /// plus an aggregate total and the canQueue flag.
    /// </summary>
    [HttpGet("queues")]
    public IActionResult GetQueues()
    {
        return Ok(new
        {
            canQueue = runner.GetCanQueue(),
            queues = new
            {
                Trade = new { count = runner.GetQueueCount(PokeRoutineType.LinkTrade) },
                SeedCheck = new { count = runner.GetQueueCount(PokeRoutineType.SeedCheck) },
                Clone = new { count = runner.GetQueueCount(PokeRoutineType.Clone) },
                Dump = new { count = runner.GetQueueCount(PokeRoutineType.Dump) },
            },
            totalCount = runner.GetTotalQueueCount(),
        });
    }
}
