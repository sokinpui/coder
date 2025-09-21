import { useState } from 'react'
import { IconButton, Tooltip, useTheme } from '@mui/material'
import { ContentCopy as ContentCopyIcon, Check as CheckIcon } from '@mui/icons-material'

interface CopyButtonProps {
  content: string;
  onCopy?: () => void;
}

export function CopyButton({ content, onCopy }: CopyButtonProps) {
  const [copied, setCopied] = useState(false)
  const theme = useTheme()

  const handleCopy = async () => {
    if (copied) return
    try {
      await navigator.clipboard.writeText(content);
      setCopied(true)
      onCopy?.();
      setTimeout(() => setCopied(false), 2000)
    } catch (err) {
      console.error('Failed to copy text: ', err)
    }
  }

  return (
    <Tooltip title={copied ? 'Copied!' : 'Copy'} placement="left" enterDelay={1000}>
      <IconButton
        onClick={handleCopy}
        size="small"
        color="inherit"
        sx={{
          backgroundColor: theme.palette.action.hover,
          '&:hover': {
            backgroundColor: theme.palette.action.selected,
          },
        }}
      >
        {copied ? <CheckIcon fontSize="small" /> : <ContentCopyIcon fontSize="small" />}
      </IconButton>
    </Tooltip>
  )
}
