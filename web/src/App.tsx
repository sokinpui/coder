import { useState, useEffect, useRef } from 'react';
import ReactMarkdown from 'react-markdown';
import './index.css';

// Define message types for better state management
interface Message {
  sender: 'User' | 'AI' | 'System' | 'Command' | 'Result' | 'Error';
  content: string;
}

function App() {
  const [messages, setMessages] = useState<Message[]>([]);
  const [input, setInput] = useState('');
  const ws = useRef<WebSocket | null>(null);
  const messagesEndRef = useRef<HTMLDivElement | null>(null);

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  };

  useEffect(scrollToBottom, [messages]);

  useEffect(() => {
    let ignore = false;

    // Initialize WebSocket connection
    const socket = new WebSocket(`ws://${location.host}/ws`);
    ws.current = socket;

    socket.onopen = () => {
      if (ignore) return;
      console.log("Connected to WebSocket");
      setMessages(prev => [...prev, { sender: 'System', content: 'Connected to server.' }]);
    };

    socket.onmessage = (event) => {
      if (ignore) return;
      const msg = JSON.parse(event.data);
      console.log("Received:", msg);

      switch (msg.type) {
        case "messageUpdate":
          setMessages(prev => [...prev, { sender: msg.payload.type, content: msg.payload.content }]);
          break;
        case "generationChunk":
          setMessages(prev => {
            const lastMessage = prev[prev.length - 1];
            if (lastMessage && lastMessage.sender === 'AI') {
              // Append to the last AI message
              const newMessages = [...prev];
              newMessages[newMessages.length - 1] = { ...lastMessage, content: lastMessage.content + msg.payload };
              return newMessages;
            } else {
              // Start a new AI message
              return [...prev, { sender: 'AI', content: msg.payload }];
            }
          });
          break;
        case "generationEnd":
          // No action needed, chunking is handled
          break;
        case "newSession":
          setMessages([{ sender: 'System', content: 'New session started.' }]);
          break;
        case "error":
          setMessages(prev => [...prev, { sender: 'Error', content: msg.payload }]);
          break;
      }
    };

    socket.onclose = () => {
      if (ignore) return;
      console.log("Connection closed");
      setMessages(prev => [...prev, { sender: 'System', content: 'Connection closed.' }]);
    };

    socket.onerror = (err) => {
      if (ignore) return;
      console.error("WebSocket error:", err);
      setMessages(prev => [...prev, { sender: 'Error', content: 'WebSocket connection error.' }]);
    };

    // Cleanup on component unmount
    return () => {
      ignore = true;
      socket.close();
    };
  }, []); // Empty dependency array means this runs once on mount

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!input.trim() || !ws.current || ws.current.readyState !== WebSocket.OPEN) {
      return;
    }

    const wsMsg = {
      type: "userInput",
      payload: input
    };
    ws.current.send(JSON.stringify(wsMsg));
    setMessages(prev => [...prev, { sender: 'User', content: input }]);
    setInput('');
  };

  return (
    <div className="app-container">
      <div className="messages-container">
        {messages.map((msg, index) => (
          <div key={index} className={`message ${msg.sender.toLowerCase()}`}>
            <strong>{msg.sender}:</strong>
            <div className="message-content">
                {msg.sender === 'AI' ? <ReactMarkdown>{msg.content}</ReactMarkdown> : <pre>{msg.content}</pre>}
            </div>
          </div>
        ))}
        <div ref={messagesEndRef} />
      </div>
      <form onSubmit={handleSubmit} className="input-form">
        <input
          type="text"
          value={input}
          onChange={(e) => setInput(e.target.value)}
          placeholder="Type your message..."
          autoComplete="off"
        />
        <button type="submit">Send</button>
      </form>
    </div>
  );
}

export default App;
