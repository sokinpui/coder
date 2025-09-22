import { Paper, IconButton, Tooltip } from '@mui/material';
import { CopyButton } from '../CopyButton';
import PsychologyIcon from '@mui/icons-material/Psychology';

interface HighlightMenuProps {
  selectedText: string;
  onCopySuccess: () => void;
  onAskAI: (text: string) => void;
}

export function HighlightMenu({ selectedText, onCopySuccess, onAskAI }: HighlightMenuProps) {
  const handleAskAI = () => {
    onAskAI(selectedText);
  };

  return (
    <Paper elevation={3} sx={{ display: 'flex', alignItems: 'center', p: 0.5, borderRadius: 1 }}>
      <CopyButton content={selectedText} onCopy={onCopySuccess} />
      <Tooltip title="Ask AI" placement="top" enterDelay={1000}>
        <IconButton onClick={handleAskAI} size="small">
          <PsychologyIcon fontSize="small" />
        </IconButton>
      </Tooltip>
    </Paper>
  );
}
