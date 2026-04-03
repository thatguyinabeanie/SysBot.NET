using System;
using Microsoft.AspNetCore.SignalR;
using SysBot.Base;

namespace SysBot.Pokemon.Web.Realtime;

public class WebLogForwarder(IHubContext<LogHub> hubContext) : ILogForwarder
{
    public void Forward(string message, string identity)
    {
        var timestamp = DateTime.Now.ToString("HH:mm:ss");
        // Fire and forget — don't block the logging pipeline
        _ = hubContext.Clients.All.SendAsync("ReceiveLog", timestamp, identity, message);
    }
}
