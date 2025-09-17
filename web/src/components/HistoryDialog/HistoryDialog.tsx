import { Dialog, DialogTitle, DialogContent, List, ListItem, ListItemButton, ListItemText, Typography } from '@mui/material'
import type { HistoryItem } from '../../types'

interface HistoryDialogProps {
  open: boolean
  onClose: () => void
  history: HistoryItem[]
  onLoad: (filename: string) => void
}

export function HistoryDialog({ open, onClose, history, onLoad }: HistoryDialogProps) {
  return (
    <Dialog open={open} onClose={onClose} fullWidth maxWidth="sm">
      <DialogTitle>Conversation History</DialogTitle>
      <DialogContent>
        {history.length === 0 ? (
          <Typography>No history found.</Typography>
        ) : (
          <List>
            {history.map((item) => (
              <ListItem key={item.filename} disablePadding>
                <ListItemButton onClick={() => onLoad(item.filename)}>
                  <ListItemText
                    primary={item.title}
                    secondary={new Date(item.modifiedAt).toLocaleString()}
                  />
                </ListItemButton>
              </ListItem>
            ))}
          </List>
        )}
      </DialogContent>
    </Dialog>
  )
}
