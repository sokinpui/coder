import { useRef, useEffect } from 'react'
import ReactMarkdown from 'react-markdown'
import { Box, Paper, Typography, CircularProgress } from '@mui/material'
import type { Message } from '../../types'
import { CopyButton } from '../CopyButton'

interface MessageListProps {
  messages: Message[]
  isGenerating: boolean
}

export function MessageList({ messages, isGenerating }: MessageListProps) {
  const scrollContainerRef = useRef<HTMLDivElement | null>(null)
  const messagesEndRef = useRef<HTMLDivElement | null>(null)

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }

  useEffect(() => {
    const container = scrollContainerRef.current
    if (!container) return

    // Only auto-scroll if the user is near the bottom.
    // This prevents the view from jumping down if they've scrolled up.
    const scrollThreshold = 100 // pixels
    const isScrolledToBottom = container.scrollHeight - container.scrollTop <= container.clientHeight + scrollThreshold

    if (isScrolledToBottom) scrollToBottom()
  }, [messages])

  return (
    <Box
      ref={scrollContainerRef}
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

        const isUser = msg.sender === 'User'
        const isError = msg.sender === 'Error'

        return (
          <Paper
            key={index}
            elevation={1}
            sx={{
              position: 'relative',
              mb: 1.5,
              maxWidth: '80%',
              alignSelf: isUser ? 'flex-end' : 'flex-start',
              bgcolor: isError ? 'error.main' : 'background.paper',
              color: isError ? 'primary.contrastText' : 'text.primary',
              overflow: 'hidden',
            }}
          >
            <Box
              sx={{
                display: 'flex',
                justifyContent: 'space-between',
                alignItems: 'center',
                position: 'sticky',
                top: 0,
                zIndex: 1,
                bgcolor: 'inherit',
                py: 0.5,
                px: 1.5,
              }}
            >
              <Typography variant="subtitle2" sx={{ fontWeight: 'bold' }}>{msg.sender}</Typography>
              <CopyButton content={msg.content} />
            </Box>
            <Box
              className="message-content"
              sx={{
                '& pre': { whiteSpace: 'pre-wrap', wordWrap: 'break-word', fontFamily: 'monospace' },
                '& code': { fontFamily: 'monospace', backgroundColor: 'action.hover', px: 0.5, borderRadius: 1 },
                '& pre > code': { display: 'block', p: 1, backgroundColor: 'action.selected' },
                px: 1.5,
                pb: 1.5,
              }}
            >
              {msg.sender === 'AI' || msg.sender === 'User' ? <ReactMarkdown>{msg.content}</ReactMarkdown> : <Typography component="pre">{msg.content}</Typography>}
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
  )
}
