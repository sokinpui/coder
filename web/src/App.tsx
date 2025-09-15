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
  CssBaseline,
  CircularProgress,
  Button,
} from '@mui/material'
import {
  Send as SendIcon,
  Brightness4 as Brightness4Icon,
  Brightness7 as Brightness7Icon,
  Menu as MenuIcon,
  AddComment as AddCommentIcon,
} from '@mui/icons-material'
import { AppContext } from './AppContext'
import { useWebSocket } from './hooks/useWebSocket'
import { Sidebar, drawerWidth, getCollapsedDrawerWidth } from './components/Sidebar'

function App() {
  const { messages, sendMessage, setMessages, cwd, isGenerating, cancelGeneration } = useWebSocket(`ws://${location.host}/ws`)
  const [input, setInput] = useState('')
  const [sidebarOpen, setSidebarOpen] = useState(false)
  const messagesEndRef = useRef<HTMLDivElement | null>(null)
  const theme = useTheme()
  const { toggleColorMode } = useContext(AppContext)
  const collapsedDrawerWidth = getCollapsedDrawerWidth(theme)

  const handleSidebarToggle = () => {
    setSidebarOpen(!sidebarOpen)
  }

  const handleNewChat = () => {
    sendMessage(':new')
  }

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }

  useEffect(scrollToBottom, [messages])

  const handleSubmit = (e: React.FormEvent | React.KeyboardEvent) => {
    e.preventDefault();
    if (!input.trim()) {
      return;
    }

    sendMessage(input);
    setMessages((prev) => [...prev, { sender: 'User', content: input }])
    setInput('')
  }

  const currentDrawerWidth = sidebarOpen ? drawerWidth : collapsedDrawerWidth

  return (
    <Box sx={{ display: 'flex', height: '100vh' }}>
      <CssBaseline />
      <AppBar
        position="fixed"
        elevation={1}
        sx={{
          zIndex: theme.zIndex.drawer + 1,
          width: `calc(100% - ${currentDrawerWidth}px)`,
          marginLeft: `${currentDrawerWidth}px`,
          transition: theme.transitions.create(['width', 'margin'], {
            easing: theme.transitions.easing.sharp,
            duration: sidebarOpen ? theme.transitions.duration.enteringScreen : theme.transitions.duration.leavingScreen,
          }),
        }}
      >
        <Toolbar variant="dense">
          <IconButton
            color="inherit"
            aria-label="open drawer"
            onClick={handleSidebarToggle}
            edge="start"
          >
            <MenuIcon />
          </IconButton>
          <Typography variant="h6" component="div" sx={{ flexGrow: 1 }}>
            Coder
          </Typography>
          <Typography variant="caption" sx={{ mr: 2, color: 'text.secondary' }}>
            {cwd}
          </Typography>
          <IconButton sx={{ ml: 1 }} onClick={toggleColorMode} color="inherit">
            {theme.palette.mode === 'dark' ? <Brightness7Icon /> : <Brightness4Icon />}
          </IconButton>
        </Toolbar>
      </AppBar>
      <Sidebar open={sidebarOpen} onNewChat={handleNewChat} isGenerating={isGenerating} />
      <Box
        component="main"
        sx={{
          flexGrow: 1,
          display: 'flex',
          flexDirection: 'column',
          height: '100vh',
          bgcolor: 'background.default',
          color: 'text.primary',
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
              )
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
            )
          })}
          {isGenerating && messages.length > 0 && messages[messages.length - 1].sender === 'User' && (
            <Paper
              elevation={1}
              sx={{
                p: 1.5,
                mb: 1.5,
                maxWidth: '80%',
                alignSelf: 'flex-start',
                bgcolor: 'background.paper',
                color: 'text.primary',
              }}
            >
              <Box sx={{ display: 'flex', alignItems: 'center' }}>
                <CircularProgress size={20} sx={{ mr: 1.5 }} />
                <Typography variant="body2">AI is thinking...</Typography>
              </Box>
            </Paper>
          )}
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
            onKeyDown={(e) => {
              if (e.key === 'Enter' && e.shiftKey) {
                e.preventDefault()
                handleSubmit(e)
              }
            }}
            placeholder="Type your message... (Enter for new line, Shift+Enter to send)"
            autoComplete="off"
            multiline
            maxRows={10}
            size="small"
            disabled={isGenerating}
            sx={{
              '& .MuiOutlinedInput-root': {
                maxHeight: '25vh',
              },
            }}
          />
          {isGenerating ? (
            <Button
              onClick={cancelGeneration}
              startIcon={<CircularProgress size={20} />}
              sx={{ ml: 1, whiteSpace: 'nowrap' }}
              variant="outlined"
              color="secondary"
            >
              Stop
            </Button>
          ) : (
            <IconButton type="submit" color="primary" sx={{ ml: 1 }} disabled={!input.trim()}>
              <SendIcon />
            </IconButton>
          )}
        </Box>
      </Box>
    </Box>
  )
}

export default App;
