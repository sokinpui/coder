import { useState } from 'react'
import { Box, TextField, IconButton, Button, CircularProgress } from '@mui/material'
import { Send as SendIcon } from '@mui/icons-material'

interface ChatInputProps {
  isGenerating: boolean
  sendMessage: (message: string) => void
  uploadImage: (dataURL: string) => void
  cancelGeneration: () => void
}

export function ChatInput({ isGenerating, sendMessage, uploadImage, cancelGeneration }: ChatInputProps) {
  const [value, setValue] = useState('')
  const handleSubmit = (e: React.FormEvent | React.KeyboardEvent) => {
    e.preventDefault()
    if (!value.trim()) {
      return
    }

    sendMessage(value)
    setValue('')
  }

  const handlePaste = (e: React.ClipboardEvent) => {
    const items = e.clipboardData.items
    for (let i = 0; i < items.length; i++) {
      if (items[i].type.indexOf('image') !== -1) {
        const file = items[i].getAsFile()
        if (file) {
          const reader = new FileReader()
          reader.onload = (event) => {
            if (event.target?.result) {
              const img = new Image()
              img.onload = () => {
                const canvas = document.createElement('canvas')
                canvas.width = img.width
                canvas.height = img.height
                const ctx = canvas.getContext('2d')
                if (ctx) {
                  ctx.drawImage(img, 0, 0)
                  const dataUrl = canvas.toDataURL('image/jpeg', 0.9) // 0.9 is quality
                  uploadImage(dataUrl)
                } else {
                  console.error('Failed to get 2D context for canvas. Image upload aborted.');
                }
              }
              img.onerror = () => {
                console.error('Failed to load image into canvas. Image upload aborted.');
              }
              img.src = event.target.result as string
            }
          }
          reader.readAsDataURL(file)
        }
        e.preventDefault() // Prevent pasting image data as text
        return
      }
    }
  }

  return (
    <Box
      component="form"
      onSubmit={handleSubmit}
      sx={{ p: 2, display: 'flex', alignItems: 'center', borderTop: 1, borderColor: 'divider', bgcolor: 'background.paper', gap: 1 }}
    >
      <TextField
        fullWidth
        variant="outlined"
        value={value}
        onChange={(e) => setValue(e.target.value)}
        onPaste={handlePaste}
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
            borderRadius: '24px',
            maxHeight: '25vh',
            '& .MuiOutlinedInput-input': {
              padding: '10px 14px',
            },
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
        <IconButton
          type="submit"
          color="primary"
          sx={{ bgcolor: 'primary.main', color: 'primary.contrastText', '&:hover': { bgcolor: 'primary.dark' } }}
          disabled={!value.trim()}
        >
          <SendIcon />
        </IconButton>
      )}
    </Box>
  )
}
