import { useState, useEffect, useRef, useCallback } from 'react';
import type { Message, HistoryItem, SourceNode, GitGraphLogEntry } from '../types';

interface AskAIParams {
  onChunk: (chunk: string) => void;
  onEnd: () => void;
  onError: (error: string) => void;
}

export function useWebSocket(url: string) {
  const [cwd, setCwd] = useState<string>('')
  const [title, setTitle] = useState<string>('New Chat')
  const [messages, setMessages] = useState<Message[]>([]);
  const [isGenerating, setIsGenerating] = useState(false);
  const [isAnimatingTitle, setIsAnimatingTitle] = useState(false);
  const [tokenCount, setTokenCount] = useState<number>(0)
  const [mode, setMode] = useState<string>('');
  const [model, setModel] = useState<string>('');
  const [availableModes, setAvailableModes] = useState<string[]>([]);
  const [availableModels, setAvailableModels] = useState<string[]>([]);
  const [history, setHistory] = useState<HistoryItem[]>([]);
  const [sourceTree, setSourceTree] = useState<SourceNode | null>(null);
  const [activeFile, setActiveFile] = useState<{ path: string; content: string } | null>(null);
  const [gitGraphLog, setGitGraphLog] = useState<GitGraphLogEntry[]>([]);
  const [commitDiff, setCommitDiff] = useState<{ hash: string; diff: string } | null>(null);
  const ws = useRef<WebSocket | null>(null);
  const fileCache = useRef<Map<string, string>>(new Map());
  const requestCallbacks = useRef<Map<string, AskAIParams>>(new Map());

  useEffect(() => {
    let ignore = false;

    const socket = new WebSocket(url);
    ws.current = socket;

    socket.onopen = () => {
      if (ignore) return;
      console.log("Connected to WebSocket");
    };

    socket.onmessage = (event) => {
      if (ignore) return;
      const msg = JSON.parse(event.data);
      console.log("Received:", msg);

      if (msg.requestId) {
        const callbacks = requestCallbacks.current.get(msg.requestId);
        if (callbacks) {
          switch (msg.type) {
            case 'generationChunk':
              callbacks.onChunk(msg.payload);
              break;
            case 'generationEnd':
              callbacks.onEnd();
              requestCallbacks.current.delete(msg.requestId);
              break;
            case 'error':
              callbacks.onError(msg.payload);
              requestCallbacks.current.delete(msg.requestId);
              break;
          }
          return; // Message handled, don't process further
        }
      }

      switch (msg.type) {
        case "initialState":
          setCwd(msg.payload.cwd || '');
          setTitle(msg.payload.title || 'New Chat');
          setIsAnimatingTitle(false);
          setTokenCount(msg.payload.tokenCount || 0)
          setMode(msg.payload.mode || '');
          setModel(msg.payload.model || '');
          setAvailableModes(msg.payload.availableModes || []);
          setAvailableModels(msg.payload.availableModels || []);
          break
        case "stateUpdate":
          setTokenCount(msg.payload.tokenCount || 0)
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
        case "truncateMessages":
          setMessages(prev => prev.slice(0, msg.payload));
          break;
        case "titleUpdate":
          setTitle(msg.payload);
          setIsAnimatingTitle(true);
          break;
        case "historyList":
          setHistory(msg.payload || []);
          break;
        case "sessionLoaded":
          setMessages(msg.payload.messages.map((m: { type: Message['sender']; content: string; }) => ({ sender: m.type, content: m.content })));
          setTitle(msg.payload.title);
          setIsAnimatingTitle(false);
          setMode(msg.payload.mode);
          setModel(msg.payload.model);
          setTokenCount(msg.payload.tokenCount);
          setIsGenerating(false);
          break;
        case "sourceTree":
          setSourceTree(msg.payload);
          break;
        case "fileContent":
          fileCache.current.set(msg.payload.path, msg.payload.content);
          setActiveFile(msg.payload);
          break;
        case "gitGraphLog":
          setGitGraphLog(msg.payload || []);
          break;
        case "commitDiff":
          setCommitDiff(msg.payload);
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

  const sendMessage = useCallback((payload: string) => {
    if (!ws.current || ws.current.readyState !== WebSocket.OPEN) {
      console.error("WebSocket is not open.");
      return;
    }
    if (payload.startsWith(':rename ')) {
        const newTitle = payload.substring(8);
        setTitle(newTitle);
        setIsAnimatingTitle(true);
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
  }, []);

  const askAI = useCallback((params: { context: string; question: string; history: Message[] } & AskAIParams) => {
    if (!ws.current || ws.current.readyState !== WebSocket.OPEN) {
      console.error("WebSocket is not open.");
      params.onError("WebSocket is not connected.");
      return;
    }

    const requestId = crypto.randomUUID();
    requestCallbacks.current.set(requestId, {
      onChunk: params.onChunk,
      onEnd: params.onEnd,
      onError: params.onError,
    });

    const wsMsg = {
      type: "askAI",
      payload: {
        context: params.context,
        question: params.question,
        history: params.history,
      },
      requestId: requestId,
    };
    ws.current.send(JSON.stringify(wsMsg));
  }, []);

  const uploadImage = useCallback((dataURL: string) => {
    if (!ws.current || ws.current.readyState !== WebSocket.OPEN) {
      console.error("WebSocket is not open.");
      return;
    }
    const wsMsg = {
      type: "imageUpload",
      payload: dataURL,
    };
    ws.current.send(JSON.stringify(wsMsg));
  }, []);

  const onTitleAnimationEnd = useCallback(() => setIsAnimatingTitle(false), []);

  const cancelGeneration = useCallback(() => {
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
  }, []);

  const regenerateFrom = useCallback((userMessageIndex: number) => {
    if (!ws.current || ws.current.readyState !== WebSocket.OPEN) {
      console.error("WebSocket is not open.");
      return;
    }
    setIsGenerating(true);
    const wsMsg = {
      type: "regenerateFrom",
      payload: userMessageIndex
    };
    ws.current.send(JSON.stringify(wsMsg));
  }, []);

  const applyItf = useCallback((content: string) => {
    if (!ws.current || ws.current.readyState !== WebSocket.OPEN) {
      console.error("WebSocket is not open.");
      return;
    }
    const wsMsg = {
      type: "applyItf",
      payload: content
    };
    ws.current.send(JSON.stringify(wsMsg));
  }, []);

  const editMessage = useCallback((index: number, content: string) => {
    if (!ws.current || ws.current.readyState !== WebSocket.OPEN) {
      console.error("WebSocket is not open.");
      return;
    }
    // Optimistic update
    setMessages(prev => {
        const newMessages = [...prev];
        if (newMessages[index] && newMessages[index].sender === 'User') {
            newMessages[index] = { ...newMessages[index], content: content };
        }
        return newMessages;
    });

    const wsMsg = {
      type: "editMessage",
      payload: { index, content }
    };
    ws.current.send(JSON.stringify(wsMsg));
  }, []);

  const branchFrom = useCallback((messageIndex: number) => {
    if (!ws.current || ws.current.readyState !== WebSocket.OPEN) {
      console.error("WebSocket is not open.");
      return;
    }
    const wsMsg = {
      type: "branchFrom",
      payload: messageIndex
    };
    ws.current.send(JSON.stringify(wsMsg));
  }, []);

  const deleteMessage = useCallback((index: number) => {
    if (!ws.current || ws.current.readyState !== WebSocket.OPEN) {
      console.error("WebSocket is not open.");
      return;
    }
    // Optimistic update
    setMessages(prev => prev.filter((_, i) => i !== index));

    const wsMsg = {
      type: "deleteMessage",
      payload: index
    };
    ws.current.send(JSON.stringify(wsMsg));
  }, []);

  const listHistory = useCallback(() => {
    if (!ws.current || ws.current.readyState !== WebSocket.OPEN) {
      console.error("WebSocket is not open.");
      return;
    }
    ws.current.send(JSON.stringify({ type: "listHistory" }));
  }, []);

  const loadConversation = useCallback((filename: string) => {
    if (!ws.current || ws.current.readyState !== WebSocket.OPEN) {
      console.error("WebSocket is not open.");
      return;
    }
    ws.current.send(JSON.stringify({ type: "loadConversation", payload: filename }));
  }, []);

  const getSourceTree = useCallback(() => {
    if (!ws.current || ws.current.readyState !== WebSocket.OPEN) {
      console.error("WebSocket is not open.");
      return;
    }
    ws.current.send(JSON.stringify({ type: "getSourceTree" }));
  }, []);

  const getFileContent = useCallback((path: string) => {
    if (fileCache.current.has(path)) {
      setActiveFile({ path, content: fileCache.current.get(path)! });
      return;
    }

    if (!ws.current || ws.current.readyState !== WebSocket.OPEN) {
      console.error("WebSocket is not open.");
      return;
    }
    ws.current.send(JSON.stringify({ type: "getFileContent", payload: path }));
  }, []);

  const clearActiveFile = useCallback(() => {
    setActiveFile(null);
  }, []);

  const getGitGraphLog = useCallback(() => {
    if (!ws.current || ws.current.readyState !== WebSocket.OPEN) {
      console.error("WebSocket is not open.");
      return;
    }
    ws.current.send(JSON.stringify({ type: "getGitGraphLog" }));
  }, []);

  const getCommitDiff = useCallback((hash: string) => {
    if (!ws.current || ws.current.readyState !== WebSocket.OPEN) {
      console.error("WebSocket is not open.");
      return;
    }
    ws.current.send(JSON.stringify({ type: "getCommitDiff", payload: hash }));
  }, []);

  return {
		messages,
		title,
    isAnimatingTitle,
    onTitleAnimationEnd,
		askAI,
		uploadImage,
		sendMessage,
		cwd,
		tokenCount,
		isGenerating,
		cancelGeneration,
		mode,
		model,
		availableModes,
		availableModels,
		regenerateFrom,
		applyItf,
		editMessage,
		branchFrom,
		deleteMessage,
		history,
		listHistory,
		loadConversation,
		sourceTree,
		getSourceTree,
		activeFile,
		getFileContent,
		clearActiveFile,
		gitGraphLog,
		getGitGraphLog,
		commitDiff,
		getCommitDiff,
	};
}
