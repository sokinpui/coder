import { useRef, useEffect, useState } from 'react'
import ReactMarkdown from 'react-markdown'
import { Box, Paper, Typography, CircularProgress, IconButton, Tooltip, TextField } from '@mui/material'
import { Replay as ReplayIcon, PlaylistAddCheck as PlaylistAddCheckIcon, Edit as EditIcon, Check as CheckIcon, Close as CloseIcon } from '@mui/icons-material'
import type { Message } from '../../types'
import { CopyButton } from '../CopyButton'

interface MessageListProps {
  messages: Message[]
  isGenerating: boolean
  onRegenerate: (userMessageIndex: number) => void
  onApplyItf: (content: string) => void
  onEditMessage: (index: number, content: string) => void
}

export function MessageList({ messages, isGenerating, onRegenerate, onApplyItf, onEditMessage }: MessageListProps) {
  const scrollContainerRef = useRef<HTMLDivElement | null>(null)
  const messagesEndRef = useRef<HTMLDivElement | null>(null)

  const [editingIndex, setEditingIndex] = useState<number | null>(null)
  const [editText, setEditText] = useState('')

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

  const handleEditStart = (index: number, content: string) => {
    setEditingIndex(index)
    setEditText(content)
  }

  const handleEditSave = () => {
    if (editingIndex !== null) {
      onEditMessage(editingIndex, editText)
    }
    setEditingIndex(null)
    setEditText('')
  }

  const handleEditCancel = () => {
    setEditingIndex(null)
    setEditText('')
  }

  return (
    <Box
      ref={scrollContainerRef}
      sx={{
        flexGrow: 1,
        overflowY: 'auto',
        px: 2,
        pb: 2,
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
        const isAI = msg.sender === 'AI'
        const isEditing = editingIndex === index

        return (
          <Paper
            key={index}
            elevation={1}
            sx={{
              position: 'relative',
              mb: 1.5,
              ...(index === 0 && { mt: 2 }),
              maxWidth: '80%',
              alignSelf: isUser ? 'flex-end' : 'flex-start',
              bgcolor: isError ? 'error.main' : 'background.paper',
              color: isError ? 'primary.contrastText' : 'text.primary',
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
                bgcolor: (theme) =>
                  theme.palette.mode === 'dark' ? theme.palette.grey[700] : theme.palette.grey[200],
                py: 0.5,
                px: 1.5,
              }}
            >
              <Typography variant="subtitle2" sx={{ fontWeight: 'bold' }}>{msg.sender}</Typography>
              <Box sx={{ display: 'flex', alignItems: 'center' }}>
                {isUser && !isGenerating && !isEditing && (
                  <Tooltip title="Edit" placement="left">
                    <IconButton
                      onClick={() => handleEditStart(index, msg.content)}
                      size="small"
                      color="inherit"
                      sx={{
                        mr: 0.5,
                        backgroundColor: (theme) => theme.palette.action.hover,
                        '&:hover': {
                          backgroundColor: (theme) => theme.palette.action.selected,
                        },
                      }}
                    >
                      <EditIcon fontSize="small" />
                    </IconButton>
                  </Tooltip>
                )}
                {isAI && !isGenerating && (
                  <Tooltip title="Apply" placement="left">
                    <IconButton
                      onClick={() => onApplyItf(msg.content)}
                      size="small"
                      color="inherit"
                      sx={{
                        mr: 0.5,
                        backgroundColor: (theme) => theme.palette.action.hover,
                        '&:hover': {
                          backgroundColor: (theme) => theme.palette.action.selected,
                        },
                      }}
                    >
                      <PlaylistAddCheckIcon fontSize="small" />
                    </IconButton>
                  </Tooltip>
                )}
                {(isUser || (isAI && index > 0 && messages[index - 1].sender === 'User')) && !isGenerating && (
                  <Tooltip title="Regenerate" placement="left">
                    <IconButton
                      onClick={() => onRegenerate(isUser ? index : index - 1)}
                      size="small"
                      color="inherit"
                      sx={{
                        mr: 0.5,
                        backgroundColor: (theme) => theme.palette.action.hover,
                        '&:hover': {
                          backgroundColor: (theme) => theme.palette.action.selected,
                        },
                      }}
                    >
                      <ReplayIcon fontSize="small" />
                    </IconButton>
                  </Tooltip>
                )}
                <CopyButton content={msg.content} />
              </Box>
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
              {isEditing ? (
                <Box sx={{ pt: 1 }}>
                  <TextField
                    fullWidth
                    multiline
                    maxRows={20}
                    value={editText}
                    onChange={(e) => setEditText(e.target.value)}
                    variant="outlined"
                    size="small"
                    autoFocus
                  />
                  <Box sx={{ display: 'flex', justifyContent: 'flex-end', mt: 1, gap: 1 }}>
                    <IconButton onClick={handleEditCancel} size="small">
                      <CloseIcon />
                    </IconButton>
                    <IconButton onClick={handleEditSave} size="small" color="primary">
                      <CheckIcon />
                    </IconButton>
                  </Box>
                </Box>
              ) : (
                <>
                  {msg.sender === 'AI' || msg.sender === 'User' ? (
                    <ReactMarkdown>{msg.content}</ReactMarkdown>
                  ) : (
                    <Typography component="pre">{msg.content}</Typography>
                  )}
                </>
              )}
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
