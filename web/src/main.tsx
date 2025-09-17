import { StrictMode, useMemo, useState } from 'react'
import { createRoot } from 'react-dom/client'
import { ThemeProvider, createTheme } from '@mui/material/styles'
import CssBaseline from '@mui/material/CssBaseline'
import './index.css'
import App from './App.tsx'
import { AppContext, type AppContextType } from './AppContext.ts'

function Root() {
  const [mode, setMode] = useState<'light' | 'dark'>('light')
  const [codeTheme, setCodeTheme] = useState<'light' | 'dark'>('light')

  const appContext = useMemo<AppContextType>(
    () => ({
      toggleColorMode: () => {
        setMode((prevMode) => (prevMode === 'light' ? 'dark' : 'light'))
      },
      codeTheme,
      toggleCodeTheme: () => {
        setCodeTheme((prevTheme) => (prevTheme === 'light' ? 'dark' : 'light'))
      },
    }),
    [codeTheme],
  )

  const theme = useMemo(
    () =>
      createTheme({
        shape: {
          borderRadius: 12,
        },
        palette: {
          mode,
          ...(mode === 'light'
            ? {
                background: {
                  default: '#f4f6f8',
                  paper: '#ffffff',
                },
              }
            : {
                background: {
                  default: '#121212',
                  paper: '#1e1e1e',
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
