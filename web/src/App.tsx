import { useState, useEffect, useRef, useContext } from 'react'
import ReactMarkdown from 'react-markdown'
import {
  Box,
  TextField,
  IconButton,
  Paper,
  Typography,
  useTheme,
  AppBar,
  Toolbar,
  Drawer,
  CssBaseline,
} from '@mui/material'
import {
  Send as SendIcon,
  Brightness4 as Brightness4Icon,
  Brightness7 as Brightness7Icon,
  Menu as MenuIcon,
} from '@mui/icons-material'
import { AppContext } from './AppContext'

// Define message types for better state management
interface Message {
  sender: 'User' | 'AI' | 'System' | 'Command' | 'Result' | 'Error';
  content: string;
}

const drawerWidth = 240

function App() {
  const [messages, setMessages] = useState<Message[]>([]);
  const [input, setInput] = useState('');
  const [sidebarOpen, setSidebarOpen] = useState(false)
  const ws = useRef<WebSocket | null>(null);
  const messagesEndRef = useRef<HTMLDivElement | null>(null);
  const theme = useTheme();
  const { toggleColorMode } = useContext(AppContext);

  const handleSidebarToggle = () => {
    setSidebarOpen(!sidebarOpen)
  }

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

  const handleSubmit = (e: React.FormEvent | React.KeyboardEvent) => {
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
    <Box sx={{ display: 'flex', height: '100vh' }}>
      <CssBaseline />
      <AppBar
        position="fixed"
        elevation={1}
        sx={{
          transition: theme.transitions.create(['margin', 'width'], {
            easing: theme.transitions.easing.sharp,
            duration: theme.transitions.duration.leavingScreen,
          }),
          ...(sidebarOpen && {
            width: `calc(100% - ${drawerWidth}px)`,
            marginLeft: `${drawerWidth}px`,
            transition: theme.transitions.create(['margin', 'width'], {
              easing: theme.transitions.easing.easeOut,
              duration: theme.transitions.duration.enteringScreen,
            }),
          }),
        }}
      >
        <Toolbar variant="dense">
          <IconButton
            color="inherit"
            aria-label="open drawer"
            onClick={handleSidebarToggle}
            edge="start"
            sx={{ mr: 2 }}
          >
            <MenuIcon />
          </IconButton>
          <Typography variant="h6" component="div" sx={{ flexGrow: 1 }}>
            Coder
          </Typography>
          <IconButton sx={{ ml: 1 }} onClick={toggleColorMode} color="inherit">
            {theme.palette.mode === 'dark' ? <Brightness7Icon /> : <Brightness4Icon />}
          </IconButton>
        </Toolbar>
      </AppBar>
      <Drawer
        sx={{
          width: drawerWidth,
          flexShrink: 0,
          '& .MuiDrawer-paper': {
            width: drawerWidth,
            boxSizing: 'border-box',
          },
        }}
        variant="persistent"
        anchor="left"
        open={sidebarOpen}
      >
        <Toolbar variant="dense" />
        <Box sx={{ overflow: 'auto' }}>
          {/* Sidebar content goes here */}
        </Box>
      </Drawer>
      <Box
        component="main"
        sx={{
          flexGrow: 1,
          display: 'flex',
          flexDirection: 'column',
          height: '100vh',
          bgcolor: 'background.default',
          color: 'text.primary',
          transition: theme.transitions.create('margin', {
            easing: theme.transitions.easing.sharp,
            duration: theme.transitions.duration.leavingScreen,
          }),
          marginLeft: `-${drawerWidth}px`,
          ...(sidebarOpen && {
            transition: theme.transitions.create('margin', {
              easing: theme.transitions.easing.easeOut,
              duration: theme.transitions.duration.enteringScreen,
            }),
            marginLeft: 0,
          }),
        }}
      >
        <Toolbar variant="dense" />
        <Box
          sx={{
            flexGrow: 1,
            overflowY: 'auto',
            p: 2,
            display: 'flex',
            flexDirection: 'column',
          }}
        >
          {messages.map((msg, index) => {
            if (msg.sender === 'System') {
              return (
                <Typography
                  key={index}
                  variant="caption"
                  sx={{ alignSelf: 'center', fontStyle: 'italic', color: 'text.secondary', mb: 1.5 }}
                >
                  {msg.content}
                </Typography>
              );
            }

            const isUser = msg.sender === 'User';
            const isError = msg.sender === 'Error';

            return (
              <Paper
                key={index}
                elevation={1}
                sx={{
                  p: 1.5,
                  mb: 1.5,
                  maxWidth: '80%',
                  alignSelf: isUser ? 'flex-end' : 'flex-start',
                  bgcolor: isUser ? 'primary.main' : isError ? 'error.main' : 'background.paper',
                  color: isUser || isError ? 'primary.contrastText' : 'text.primary',
                }}
              >
                <Typography variant="subtitle2" sx={{ fontWeight: 'bold', mb: 0.5 }}>
                  {msg.sender}
                </Typography>
                <Box
                  className="message-content"
                  sx={{
                    '& pre': { whiteSpace: 'pre-wrap', wordWrap: 'break-word', fontFamily: 'monospace' },
                    '& code': { fontFamily: 'monospace', backgroundColor: 'action.hover', px: 0.5, borderRadius: 1 },
                    '& pre > code': { display: 'block', p: 1, backgroundColor: 'action.selected' },
                  }}
                >
                  {msg.sender === 'AI' ? <ReactMarkdown>{msg.content}</ReactMarkdown> : <Typography component="pre">{msg.content}</Typography>}
                </Box>
              </Paper>
            );
          })}
          <div ref={messagesEndRef} />
        </Box>
        <Box
          component="form"
          onSubmit={handleSubmit}
          sx={{ p: 1, display: 'flex', alignItems: 'flex-end', borderTop: 1, borderColor: 'divider', bgcolor: 'background.paper' }}
        >
          <TextField
            fullWidth
            variant="outlined"
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={(e) => { if (e.key === 'Enter' && e.shiftKey) { e.preventDefault(); handleSubmit(e); } }}
            placeholder="Type your message... (Enter for new line, Shift+Enter to send)"
            autoComplete="off"
            multiline
            maxRows={10}
            size="small"
            sx={{
              '& .MuiOutlinedInput-root': {
                maxHeight: '25vh',
              },
            }}
          />
          <IconButton type="submit" color="primary" sx={{ ml: 1 }}>
            <SendIcon />
          </IconButton>
        </Box>
      </Box>
    </Box>
  );
}

export default App;
