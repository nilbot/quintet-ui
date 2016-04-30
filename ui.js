var ws;
function init() {
  console.log("init.");
  var mail = document.getElementById("mail");
  var status = document.getElementById("status");
  var numClients = 0;
  var numSenders = 0;
  function onOpen() {
     status.innerText = "connected";
  };
  function onClose() {
     status.innerHTML = "connection closed";
  };
  function updateStatus() {
     status.innerHTML = "connected<br>(" + numClients + " browsers, " + numSenders + " SMTP clients)";
     return;
  }
  function onMessage(e) {
     var m = JSON.parse(e.data);
     if (m.NumClients != null) {
         numClients = m.NumClients;
         updateStatus();
         return;
     }
     if (m.NumSenders != null) {
         numSenders = m.NumSenders;
         updateStatus();
         return;
     }
     console.log(m);
     var md = document.createElement("div");
     md.innerHTML = "<table>" +
		"<tr class='from'><th>From:</th><td>" + m.From + " <i>[could be fake]</i></td></tr>" +
		"<tr class='to'><th>To:</th><td>"+m.To+"</td></tr>" +
		"<tr class='subject'><th>Subject:</th><td>" + m.Subject + "</td></tr>" +
		"<tr class='body'><th>Body:</th><td>" + m.Body + "</td></tr>" +
                "</table>";
     mail.insertBefore(md, mail.firstChild)
  };
  function connect() {
    if (ws != null) {
       ws.close();
       ws = null;
    }
    status.innerText = "connecting...";
    var url = "ws://{{.WSAddr}}/stream";
    ws = new WebSocket(url);
    ws.onopen = onOpen;
    ws.onclose = onClose;
    ws.onmessage = onMessage;
  }
  connect();
}
