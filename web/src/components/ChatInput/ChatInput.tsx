import { useState } from 'react'
import { Box, TextField, IconButton, Button, CircularProgress } from '@mui/material'
import { Send as SendIcon } from '@mui/icons-material'

interface ChatInputProps {
  isGenerating: boolean
  sendMessage: (message: string) => void
  cancelGeneration: () => void
}

export function ChatInput({ isGenerating, sendMessage, cancelGeneration }: ChatInputProps) {
  const [input, setInput] = useState('')

  const handleSubmit = (e: React.FormEvent | React.KeyboardEvent) => {
    e.preventDefault()
    if (!input.trim()) {
      return
    }

    sendMessage(input)
    setInput('')
  }

  return (
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
  )
}
