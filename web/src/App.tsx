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
  List,
  ListItem,
  ListItemButton,
  ListItemIcon,
  ListItemText,
  Divider,
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
import type { Message } from './types'

const drawerWidth = 240

function App() {
  const { messages, sendMessage, setMessages } = useWebSocket(`ws://${location.host}/ws`)
  const [input, setInput] = useState('')
  const [sidebarOpen, setSidebarOpen] = useState(false)
  const messagesEndRef = useRef<HTMLDivElement | null>(null)
  const theme = useTheme()
  const { toggleColorMode } = useContext(AppContext)
  const collapsedDrawerWidth = theme.spacing(7)

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
          <IconButton sx={{ ml: 1 }} onClick={toggleColorMode} color="inherit">
            {theme.palette.mode === 'dark' ? <Brightness7Icon /> : <Brightness4Icon />}
          </IconButton>
        </Toolbar>
      </AppBar>
      <Drawer
        variant="permanent"
        anchor="left"
        sx={{
          width: currentDrawerWidth,
          flexShrink: 0,
          whiteSpace: 'nowrap',
          boxSizing: 'border-box',
          '& .MuiDrawer-paper': {
            width: currentDrawerWidth,
            transition: theme.transitions.create('width', {
              easing: theme.transitions.easing.sharp,
              duration: sidebarOpen ? theme.transitions.duration.enteringScreen : theme.transitions.duration.leavingScreen,
            }),
            overflowX: 'hidden',
            boxSizing: 'border-box',
          },
        }}
      >
        <Toolbar variant="dense" />
        <Box>
          <List>
            <ListItem disablePadding sx={{ display: 'block' }}>
              <ListItemButton
                onClick={handleNewChat}
                sx={{
                  minHeight: 48,
                  justifyContent: sidebarOpen ? 'initial' : 'center',
                  px: 2.5,
                }}
              >
                <ListItemIcon
                  sx={{
                    minWidth: 0,
                    mr: sidebarOpen ? 3 : 'auto',
                    justifyContent: 'center',
                  }}
                >
                  <AddCommentIcon />
                </ListItemIcon>
                <ListItemText primary="New Chat" sx={{ opacity: sidebarOpen ? 1 : 0 }} />
              </ListItemButton>
            </ListItem>
          </List>
          <Divider />
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
  )
}

export default App;
