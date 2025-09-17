import { useContext } from 'react'
import {
  AppBar,
  Toolbar,
  IconButton,
  Box,
  Typography,
  Divider,
  FormControl,
  Select,
  MenuItem,
  useTheme,
  type SelectChangeEvent,
} from '@mui/material'
import {
  Menu as MenuIcon,
  Brightness4 as Brightness4Icon,
  Brightness7 as Brightness7Icon,
} from '@mui/icons-material'
import { AppContext } from '../../AppContext'

interface TopBarProps {
  onSidebarToggle: () => void
  title: string
  tokenCount: number
  cwd: string
  mode: string
  onModeChange: (event: SelectChangeEvent) => void
  availableModes: string[]
  model: string
  onModelChange: (event: SelectChangeEvent) => void
  availableModels: string[]
  isGenerating: boolean
}

export function TopBar({
  onSidebarToggle,
  title,
  tokenCount,
  cwd,
  mode,
  onModeChange,
  availableModes,
  model,
  onModelChange,
  availableModels,
  isGenerating,
}: TopBarProps) {
  const theme = useTheme()
  const { toggleColorMode } = useContext(AppContext)

  return (
    <AppBar position="static" elevation={1}>
      <Toolbar variant="dense">
        <IconButton
          color="inherit"
          aria-label="open drawer"
          onClick={onSidebarToggle}
          edge="start"
        >
          <MenuIcon />
        </IconButton>
        <Typography variant="h6" noWrap component="div" sx={{ ml: 1, mr: 2 }}>
          {title}
        </Typography>
        <Box sx={{ flexGrow: 1 }} />
        <Typography variant="body2" sx={{ color: 'inherit' }}>
          {`Tokens: ${tokenCount}`}
        </Typography>

        <Divider orientation="vertical" flexItem sx={{ mx: 1.5, my: 1, borderColor: 'rgba(255, 255, 255, 0.2)' }} />

        <Typography variant="body2" sx={{ color: 'inherit' }}>
          {cwd}
        </Typography>

        <Divider orientation="vertical" flexItem sx={{ mx: 1.5, my: 1, borderColor: 'rgba(255, 255, 255, 0.2)' }} />

        <FormControl size="small" variant="standard" sx={{ minWidth: 120 }} disabled={isGenerating}>
          <Select
            value={mode}
            onChange={onModeChange}
            disableUnderline
            sx={{ color: 'inherit', '& .MuiSelect-icon': { color: 'inherit' } }}
          >
            {availableModes.map((m) => (
              <MenuItem key={m} value={m}>{m}</MenuItem>
            ))}
          </Select>
        </FormControl>

        <Divider orientation="vertical" flexItem sx={{ mx: 1.5, my: 1, borderColor: 'rgba(255, 255, 255, 0.2)' }} />

        <FormControl size="small" variant="standard" sx={{ minWidth: 200 }} disabled={isGenerating}>
          <Select
            value={model}
            onChange={onModelChange}
            disableUnderline
            sx={{ color: 'inherit', '& .MuiSelect-icon': { color: 'inherit' } }}
          >
            {availableModels.map((m) => (
              <MenuItem key={m} value={m}>{m}</MenuItem>
            ))}
          </Select>
        </FormControl>
        <IconButton sx={{ ml: 1 }} onClick={toggleColorMode} color="inherit">
          {theme.palette.mode === 'dark' ? <Brightness7Icon /> : <Brightness4Icon />}
        </IconButton>
      </Toolbar>
    </AppBar>
  )
}
