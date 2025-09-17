import { useState, useEffect } from 'react';
import {
  Box,
  CssBaseline,
	type SelectChangeEvent,
} from '@mui/material'
import { useWebSocket } from './hooks/useWebSocket'
import { Routes, Route, useNavigate, useLocation } from 'react-router-dom';
import { Sidebar } from './components/Sidebar'
import { MessageList } from './components/MessageList'
import { ChatInput } from './components/ChatInput'
import { TopBar } from './components/TopBar';
import { HistoryDialog } from './components/HistoryDialog';
import { RenameDialog } from './components/RenameDialog';
import { SourceBrowser } from './components/SourceBrowser';
import { GitBrowser } from './components/GitBrowser';

function App() {
  const {
		messages,
		title,
		sendMessage,
		cwd,
		isGenerating,
		tokenCount,
		cancelGeneration,
		mode,
		regenerateFrom,
		applyItf,
		model,
		editMessage,
		branchFrom,
		deleteMessage,
		availableModes,
		availableModels,
		history,
		listHistory,
		loadConversation,
		sourceTree,
		getSourceTree,
		activeFile,
		getFileContent,
		clearActiveFile,
		gitGraphLog,
		getGitGraphLog,
		commitDiff,
		getCommitDiff,
	} = useWebSocket(`ws://${window.location.host}/ws`)
	const [sidebarOpen, setSidebarOpen] = useState(false)
	const [historyDialogOpen, setHistoryDialogOpen] = useState(false)
	const [renameDialogOpen, setRenameDialogOpen] = useState(false)
	const [inputValue, setInputValue] = useState('')
	const [showLineNumbers, setShowLineNumbers] = useState(false)

  const navigate = useNavigate();
  const location = useLocation();

  const pathParts = location.pathname.split('/').filter(Boolean);
  let view: 'chat' | 'code' | 'git' = 'chat';
  const firstPart = pathParts[0];
  if (firstPart === 'code' || firstPart === 'git') {
    view = firstPart;
  }

  const handleSidebarToggle = () => {
    setSidebarOpen(!sidebarOpen)
  }

  const handleNewChat = () => {
    sendMessage(':new')
    setInputValue('')
		navigate('/');
  }

  const handleChatViewOpen = () => {
    navigate('/');
  }

  const handleHistoryOpen = () => {
    listHistory()
    setHistoryDialogOpen(true)
  }

  const handleHistoryClose = () => {
    setHistoryDialogOpen(false)
  }

  const handleLoadConversation = (filename: string) => {
    loadConversation(filename)
    handleHistoryClose()
    navigate('/');
  }

  const handleRenameOpen = () => {
    setRenameDialogOpen(true)
  }

  const handleRenameSave = (newTitle: string) => {
    sendMessage(`:rename ${newTitle}`)
    setRenameDialogOpen(false)
  }

  const handleSourceBrowserOpen = () => {
    if (!sourceTree) {
      getSourceTree()
    }
		navigate('/code');
  }

  const handleGitBrowserOpen = () => {
    getGitGraphLog()
    navigate('/git');
  }

	const handleToggleLineNumbers = () => {
		setShowLineNumbers((prev) => !prev)
	}

	const handleModeChange = (event: SelectChangeEvent) => {
		sendMessage(`:mode ${event.target.value}`)
	}

	const handleModelChange = (event: SelectChangeEvent) => {
		sendMessage(`:model ${event.target.value}`)
  }

  const handleSendMessage = (message: string) => {
    sendMessage(message)
    setInputValue('')
  }

  const handleRegenerate = (index: number) => {
    regenerateFrom(index)
  }

  const handleApplyItf = (content: string) => {
    applyItf(content)
  }

  const handleEditMessage = (index: number, content: string) => {
    editMessage(index, content)
  }

  const handleBranchFrom = (index: number) => {
    branchFrom(index)
  }

  const handleDeleteMessage = (index: number) => {
    deleteMessage(index)
  }

  const handleFileSelect = (path: string) => {
    navigate(`/code/${path}`);
  };

  // Effect for deep linking into code browser and showing README by default
  useEffect(() => {
    const path = location.pathname;
    if (path.startsWith('/code')) {
      const filePath = path.startsWith('/code/') ? path.substring('/code/'.length) : '';

      if (filePath) {
        if (!activeFile || activeFile.path !== filePath) {
          getFileContent(filePath);
        }
      } else { // No file path in URL, e.g., /code or /code/
        clearActiveFile();
        if (sourceTree) {
          // Find README.md case-insensitively
          const readmeFile = sourceTree.children?.find(
            (node) => node.name.toLowerCase() === 'readme.md'
          );
          if (readmeFile) {
            // Navigate to the README file, which will trigger this effect again to load it.
            navigate(`/code/${readmeFile.path}`, { replace: true });
          }
        }
      }
    }
  }, [location.pathname, getFileContent, activeFile, sourceTree, navigate, clearActiveFile]);

  // Effect for deep linking into git browser
  useEffect(() => {
    const path = location.pathname;
    if (path.startsWith('/git/')) {
      const hash = path.substring('/git/'.length);
      if (hash && (!commitDiff || commitDiff.hash !== hash)) {
        getCommitDiff(hash);
      }
    }
  }, [location.pathname, getCommitDiff, commitDiff]);

  return (
    <Box sx={{ display: 'flex', height: '100vh' }}>
      <CssBaseline />
      <Sidebar
        open={sidebarOpen}
        onNewChat={handleNewChat}
        isGenerating={isGenerating}
        onHistoryOpen={handleHistoryOpen}
        onChatViewOpen={handleChatViewOpen}
        onCodeBrowserOpen={handleSourceBrowserOpen}
        onGitBrowserOpen={handleGitBrowserOpen}
      />
      <Box
        component="main"
        sx={{
          flexGrow: 1,
          display: 'flex',
          flexDirection: 'column',
          height: '100vh',
          bgcolor: 'background.default',
          color: 'text.primary',
          minWidth: 0, // Prevents the content from overflowing when the sidebar is open
        }}
      >
        <TopBar
          onSidebarToggle={handleSidebarToggle}
					view={view}
					title={view === 'code' ? 'Code' : view === 'git' ? 'Git' : title}
          onTitleRename={handleRenameOpen}
          tokenCount={tokenCount}
          cwd={cwd}
          mode={mode}
          onModeChange={handleModeChange}
          availableModes={availableModes}
          model={model}
          onModelChange={handleModelChange}
          availableModels={availableModels}
          isGenerating={isGenerating}
					showLineNumbers={showLineNumbers}
					onToggleLineNumbers={handleToggleLineNumbers}
        />
        <Routes>
          <Route path="/code/*" element={
            <SourceBrowser
              tree={sourceTree}
              activeFile={activeFile}
              onFileSelect={handleFileSelect}
              showLineNumbers={showLineNumbers}
            />
          } />
          <Route path="/git/*" element={
            <GitBrowser log={gitGraphLog} commitDiff={commitDiff} />
          } />
          <Route path="/*" element={
            <>
              <MessageList
                messages={messages}
                isGenerating={isGenerating}
                onRegenerate={handleRegenerate}
                onApplyItf={handleApplyItf}
                onEditMessage={handleEditMessage}
                onBranchFrom={handleBranchFrom}
                onDeleteMessage={handleDeleteMessage}
              />
              <ChatInput sendMessage={handleSendMessage} cancelGeneration={cancelGeneration} isGenerating={isGenerating} value={inputValue} onChange={setInputValue} />
            </>
          } />
        </Routes>
      </Box>
      <HistoryDialog
        open={historyDialogOpen}
        onClose={handleHistoryClose}
        history={history}
        onLoad={handleLoadConversation}
      />
      <RenameDialog
        open={renameDialogOpen}
        onClose={() => setRenameDialogOpen(false)}
        onSave={handleRenameSave}
        currentTitle={title}
      />
    </Box>
  )
}

export default App;
