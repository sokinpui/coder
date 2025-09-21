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
  Edit as EditIcon,
  Palette as PaletteIcon,
  FormatListNumbered as FormatListNumberedIcon,
  Refresh as RefreshIcon,
} from '@mui/icons-material'
import { AppContext } from '../../AppContext'

interface TopBarProps {
  onSidebarToggle: () => void
  title: string
  onTitleRename: () => void
  tokenCount: number
  cwd: string
  mode: string
  onModeChange: (event: SelectChangeEvent) => void
  availableModes: string[]
  model: string
  onModelChange: (event: SelectChangeEvent) => void
  availableModels: string[]
  isGenerating: boolean
	view: 'chat' | 'code' | 'git'
	showLineNumbers: boolean
	onToggleLineNumbers: () => void
	onReload: () => void
}

export function TopBar({
  onSidebarToggle,
  title,
  onTitleRename,
  tokenCount,
  cwd,
  mode,
  onModeChange,
  availableModes,
  model,
  onModelChange,
  availableModels,
  isGenerating,
	view,
	showLineNumbers,
	onToggleLineNumbers,
	onReload,
}: TopBarProps) {
  const theme = useTheme()
  const { toggleColorMode, toggleCodeTheme } = useContext(AppContext)

  return (
    <AppBar position="static" elevation={0} sx={{ borderBottom: 1, borderColor: 'divider' }}>
      <Toolbar variant="dense">
        <IconButton
          color="inherit"
          aria-label="open drawer"
          onClick={onSidebarToggle}
          edge="start"
        >
          <MenuIcon />
        </IconButton>
        <Box
          sx={{
            display: 'flex',
            alignItems: 'center',
            ml: 1,
            mr: 2,
            '&:hover .rename-button': {
              opacity: 1,
            },
          }}
        >
          <Typography variant="subtitle1" noWrap component="div" sx={{ fontWeight: 'bold' }}>
            {title}
          </Typography>
					{view === 'chat' && (
						<IconButton onClick={onTitleRename} size="small" className="rename-button" sx={{ ml: 0.5, opacity: 0, transition: 'opacity 0.2s' }}>
							<EditIcon fontSize="small" />
						</IconButton>
					)}
        </Box>
        <Box sx={{ flexGrow: 1 }} />
				{view === 'chat' && (
					<>
						<Typography variant="body2" sx={{ color: 'inherit', display: { xs: 'none', md: 'block' } }}>
							{`Tokens: ${tokenCount}`}
						</Typography>

						<Divider orientation="vertical" flexItem sx={{ mx: 1.5, my: 1, borderColor: 'rgba(255, 255, 255, 0.2)', display: { xs: 'none', md: 'block' } }} />

						<Typography variant="body2" sx={{ color: 'inherit', display: { xs: 'none', lg: 'block' } }}>
							{cwd}
						</Typography>

						<Divider orientation="vertical" flexItem sx={{ mx: 1.5, my: 1, borderColor: 'rgba(255, 255, 255, 0.2)', display: { xs: 'none', lg: 'block' } }} />

						<FormControl size="small" sx={{ minWidth: { xs: 100, sm: 120 } }} disabled={isGenerating}>
							<Select
								value={mode}
								onChange={onModeChange}
								sx={{
									color: 'inherit',
									'.MuiOutlinedInput-notchedOutline': { borderColor: 'rgba(255, 255, 255, 0.23)' },
									'&.Mui-focused .MuiOutlinedInput-notchedOutline': { borderColor: 'inherit' },
									'&:hover .MuiOutlinedInput-notchedOutline': { borderColor: 'rgba(255, 255, 255, 0.5)' },
									'.MuiSvgIcon-root': { color: 'inherit' },
									borderRadius: '20px',
								}}
							>
								{availableModes.map((m) => (
									<MenuItem key={m} value={m}>{m}</MenuItem>
								))}
							</Select>
						</FormControl>

						<Divider orientation="vertical" flexItem sx={{ mx: 1.5, my: 1, borderColor: 'rgba(255, 255, 255, 0.2)' }} />

						<FormControl size="small" sx={{ minWidth: { xs: 120, sm: 160, md: 200 } }} disabled={isGenerating}>
							<Select
								value={model}
								onChange={onModelChange}
								sx={{
									color: 'inherit',
									'.MuiOutlinedInput-notchedOutline': { borderColor: 'rgba(255, 255, 255, 0.23)' },
									'&.Mui-focused .MuiOutlinedInput-notchedOutline': { borderColor: 'inherit' },
									'&:hover .MuiOutlinedInput-notchedOutline': { borderColor: 'rgba(255, 255, 255, 0.5)' },
									'.MuiSvgIcon-root': { color: 'inherit' },
									borderRadius: '20px',
								}}
							>
								{availableModels.map((m) => (
									<MenuItem key={m} value={m}>{m}</MenuItem>
								))}
							</Select>
						</FormControl>
					</>
				)}
        {(view === 'code' || view === 'git') && (
          <IconButton onClick={onReload} color="inherit">
            <RefreshIcon />
          </IconButton>
        )}
        {view === 'code' && (
          <IconButton onClick={onToggleLineNumbers} color={showLineNumbers ? 'secondary' : 'inherit'}>
            <FormatListNumberedIcon />
          </IconButton>
        )}
        <IconButton sx={{ ml: 1 }} onClick={toggleCodeTheme} color="inherit">
          <PaletteIcon />
        </IconButton>
        <IconButton sx={{ ml: 1 }} onClick={toggleColorMode} color="inherit">
          {theme.palette.mode === 'dark' ? <Brightness7Icon /> : <Brightness4Icon />}
        </IconButton>
      </Toolbar>
    </AppBar>
  )
}
