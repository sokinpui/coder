import { useState, useEffect, useRef } from 'react';
import type { Message } from '../types';

export function useWebSocket(url: string) {
  const [cwd, setCwd] = useState<string>('')
  const [messages, setMessages] = useState<Message[]>([]);
  const [isGenerating, setIsGenerating] = useState(false);
  const [mode, setMode] = useState<string>('');
  const [model, setModel] = useState<string>('');
  const [availableModes, setAvailableModes] = useState<string[]>([]);
  const [availableModels, setAvailableModels] = useState<string[]>([]);
  const ws = useRef<WebSocket | null>(null);

  useEffect(() => {
    let ignore = false;

    const socket = new WebSocket(url);
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
        case "initialState":
          setCwd(msg.payload.cwd || '');
          setMode(msg.payload.mode || '');
          setModel(msg.payload.model || '');
          setAvailableModes(msg.payload.availableModes || []);
          setAvailableModels(msg.payload.availableModels || []);
          break
        case "stateUpdate":
          setMode(msg.payload.mode);
          setModel(msg.payload.model);
          break;
        case "messageUpdate":
          setMessages(prev => [...prev, { sender: msg.payload.type, content: msg.payload.content }]);
          break;
        case "generationChunk":
          setMessages(prev => {
            const lastMessage = prev[prev.length - 1];
            if (lastMessage && lastMessage.sender === 'AI') {
              const newMessages = [...prev];
              newMessages[newMessages.length - 1] = { ...lastMessage, content: lastMessage.content + msg.payload };
              return newMessages;
            } else {
              return [...prev, { sender: 'AI', content: msg.payload }];
            }
          });
          break;
        case "generationEnd":
          // No action needed, chunking is handled
          setIsGenerating(false);
          break;
        case "newSession":
          setMessages([{ sender: 'System', content: 'New session started.' }]);
          setIsGenerating(false);
          break;
        case "error":
          setMessages(prev => [...prev, { sender: 'Error', content: msg.payload }]);
          setIsGenerating(false);
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

    return () => {
      ignore = true;
      socket.close();
    };
  }, [url]);

  const sendMessage = (payload: string) => {
    if (!ws.current || ws.current.readyState !== WebSocket.OPEN) {
      console.error("WebSocket is not open.");
      return;
    }
    if (!payload.startsWith(':')) {
      setIsGenerating(true);
      // Optimistic update for user messages
      setMessages((prev) => [...prev, { sender: 'User', content: payload }])
    }
    const wsMsg = {
      type: "userInput",
      payload: payload
    };
    ws.current.send(JSON.stringify(wsMsg));
  };

  const cancelGeneration = () => {
    if (!ws.current || ws.current.readyState !== WebSocket.OPEN) {
      console.error("WebSocket is not open.");
      return;
    }
    const wsMsg = {
      type: "cancelGeneration",
      payload: ""
    };
    ws.current.send(JSON.stringify(wsMsg));
    setIsGenerating(false);
  };


  return {
		messages,
		sendMessage,
		cwd,
		isGenerating,
		cancelGeneration,
		mode,
		model,
		availableModes,
		availableModels,
	};
}
