using System;

namespace SysBot.Pokemon.Web.Models;

public class BotDto
{
    public required string Id { get; set; }
    public required string Ip { get; set; }
    public int Port { get; set; }
    public required string Protocol { get; set; }
    public required string InitialRoutine { get; set; }
    public required string CurrentRoutine { get; set; }
    public required string NextRoutine { get; set; }
    public bool IsRunning { get; set; }
    public bool IsPaused { get; set; }
    public bool IsConnected { get; set; }
    public string? LastLog { get; set; }
    public DateTime? LastActive { get; set; }
}

public class AddBotRequest
{
    public required string Ip { get; set; }
    public int Port { get; set; } = 6000;
    public required string Protocol { get; set; }
    public required string Routine { get; set; }
}
