import { useState } from 'react'
import { Box, TextField, IconButton, Button, CircularProgress } from '@mui/material'
import { Send as SendIcon } from '@mui/icons-material'

interface ChatInputProps {
  isGenerating: boolean
  sendMessage: (message: string) => void
  cancelGeneration: () => void
}

export function ChatInput({ isGenerating, sendMessage, cancelGeneration }: ChatInputProps) {
  const [value, setValue] = useState('')
  const handleSubmit = (e: React.FormEvent | React.KeyboardEvent) => {
    e.preventDefault()
    if (!value.trim()) {
      return
    }

    sendMessage(value)
    setValue('')
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
