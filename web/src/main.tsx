import { StrictMode, useMemo, useState } from 'react'
import { createRoot } from 'react-dom/client'
import { ThemeProvider, createTheme } from '@mui/material/styles'
import CssBaseline from '@mui/material/CssBaseline'
import './index.css'
import App from './App.tsx'
import { AppContext, type AppContextType } from './AppContext.ts'

function Root() {
  const [mode, setMode] = useState<'light' | 'dark'>('light')

  const appContext = useMemo<AppContextType>(
    () => ({
      toggleColorMode: () => {
        setMode((prevMode) => (prevMode === 'light' ? 'dark' : 'light'))
      },
    }),
    [],
  )

  const theme = useMemo(
    () =>
      createTheme({
        palette: {
          mode,
          ...(mode === 'light'
            ? {
                // Use a light grey background for light mode to reduce eye strain
                background: {
                  default: '#fafafa',
                  paper: '#fff',
                },
              }
            : {
                // Custom dark mode colors for better contrast
                background: {
                  default: '#303030',
                  paper: '#424242',
                },
              }),
        },
      }),
    [mode],
  )

  return (
    <AppContext.Provider value={appContext}>
      <ThemeProvider theme={theme}>
        <CssBaseline />
        <App />
      </ThemeProvider>
    </AppContext.Provider>
  )
}

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <Root />
  </StrictMode>,
)
