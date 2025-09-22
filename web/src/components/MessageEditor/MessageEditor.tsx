import { useState } from 'react';
import { Box, TextField, IconButton } from '@mui/material';
import { Check as CheckIcon, Close as CloseIcon } from '@mui/icons-material';

interface MessageEditorProps {
  initialContent: string;
  onSave: (newContent: string) => void;
  onCancel: () => void;
}

export function MessageEditor({ initialContent, onSave, onCancel }: MessageEditorProps) {
  const [content, setContent] = useState(initialContent);

  const handleSave = () => {
    onSave(content);
  };

  return (
    <Box sx={{ pt: 1 }}>
      <TextField
        fullWidth
        multiline
        maxRows={20}
        value={content}
        onChange={(e) => setContent(e.target.value)}
        variant="outlined"
        size="small"
        autoFocus
      />
      <Box sx={{ display: 'flex', justifyContent: 'flex-end', mt: 1, gap: 1 }}>
        <IconButton onClick={onCancel} size="small">
          <CloseIcon />
        </IconButton>
        <IconButton onClick={handleSave} size="small" color="primary">
          <CheckIcon />
        </IconButton>
      </Box>
    </Box>
  );
}
