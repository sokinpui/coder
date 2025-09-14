window.addEventListener("load", function(evt) {
    const messages = document.getElementById("messages");
    const form = document.getElementById("input-form");
    const input = document.getElementById("input");
    let currentAIMessageElement = null;

    const ws = new WebSocket(`ws://${location.host}/ws`);

    ws.onopen = function() {
        console.log("Connected to WebSocket");
        addMessage("System", "Connected to server.");
    };

    ws.onmessage = function(event) {
        const msg = JSON.parse(event.data);
        console.log("Received:", msg);

        switch (msg.type) {
            case "messageUpdate":
                addMessage(msg.payload.type, msg.payload.content);
                break;
            case "generationChunk":
                if (!currentAIMessageElement) {
                    currentAIMessageElement = addMessage("AI", "");
                }
                currentAIMessageElement.textContent += msg.payload;
                break;
            case "generationEnd":
                currentAIMessageElement = null;
                break;
            case "newSession":
                messages.innerHTML = "";
                addMessage("System", "New session started.");
                break;
            case "error":
                 addMessage("Error", msg.payload);
                 break;
        }
        messages.scrollTop = messages.scrollHeight;
    };

    ws.onclose = function() {
        console.log("Connection closed");
        addMessage("System", "Connection closed.");
    };

    ws.onerror = function(err) {
        console.error("WebSocket error:", err);
        addMessage("Error", "WebSocket connection error.");
    };

    form.addEventListener("submit", function(event) {
        event.preventDefault();
        const message = input.value;
        if (!message) {
            return;
        }

        const wsMsg = {
            type: "userInput",
            payload: message
        };
        ws.send(JSON.stringify(wsMsg));
        addMessage("User", message);
        input.value = "";
    });

    function addMessage(sender, text) {
        const div = document.createElement("div");
        div.className = "message";
        
        const content = document.createElement("pre");
        content.textContent = text;
        
        div.innerHTML = `<strong>${sender}:</strong>`;
        div.appendChild(content);
        
        messages.appendChild(div);
        return content;
    }
});
