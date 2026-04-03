using Microsoft.AspNetCore.SignalR;

namespace SysBot.Pokemon.Web.Realtime;

// Client methods:
// - ReceiveLog(string timestamp, string identity, string message)
// - ReceiveEcho(string timestamp, string message)
// - BotStatusChanged(object bot)
public class LogHub : Hub
{
    // Method name constants for SignalR client invocations.
    public const string ReceiveLog = "ReceiveLog";
    public const string ReceiveEcho = "ReceiveEcho";
    public const string BotStatusChanged = "BotStatusChanged";
}
