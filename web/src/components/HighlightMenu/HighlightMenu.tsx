import { Paper } from '@mui/material';
import { CopyButton } from '../CopyButton';

interface HighlightMenuProps {
  selectedText: string;
  onCopySuccess: () => void;
}

export function HighlightMenu({ selectedText, onCopySuccess }: HighlightMenuProps) {
  return (
    <Paper elevation={3} sx={{ display: 'flex', alignItems: 'center', p: 0.5, borderRadius: 1 }}>
      <CopyButton content={selectedText} onCopy={onCopySuccess} />
    </Paper>
  );
}
