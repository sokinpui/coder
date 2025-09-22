import { useState, useEffect, useCallback } from 'react';
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
import { FloatingChatWindow } from './components/FloatingChatWindow';

function App() {
  const {
		messages,
		title: finalTitle,
    isAnimatingTitle,
    onTitleAnimationEnd,
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
		askAI,
	} = useWebSocket(`ws://${window.location.host}/ws`)
	const [sidebarOpen, setSidebarOpen] = useState(false)
	const [historyDialogOpen, setHistoryDialogOpen] = useState(false)
  const [renameDialogOpen, setRenameDialogOpen] = useState(false)
  const [displayedTitle, setDisplayedTitle] = useState(finalTitle)
	const [sessionKey, setSessionKey] = useState(() => Date.now().toString())
	const [floatingChat, setFloatingChat] = useState<{ open: boolean; context: string }>({ open: false, context: '' });
	const [showLineNumbers, setShowLineNumbers] = useState(false)

  useEffect(() => {
    if (isAnimatingTitle) {
      setDisplayedTitle(''); // Reset for animation
      let i = 0;
      const interval = setInterval(() => {
        i++;
        if (i > finalTitle.length) {
          clearInterval(interval);
          onTitleAnimationEnd();
        } else {
          setDisplayedTitle(finalTitle.substring(0, i));
        }
      }, 50);
      return () => clearInterval(interval);
    } else {
      setDisplayedTitle(finalTitle);
    }
  }, [finalTitle, isAnimatingTitle, onTitleAnimationEnd]);

  const navigate = useNavigate();
  const location = useLocation();

  const pathParts = location.pathname.split('/').filter(Boolean);
  let view: 'chat' | 'code' | 'git' = 'chat';
  const firstPart = pathParts[0];
  if (firstPart === 'code' || firstPart === 'git') {
    view = firstPart;
  }

  const handleSidebarToggle = useCallback(() => {
    setSidebarOpen(prev => !prev)
  }, []);

  const handleNewChat = useCallback(() => {
    sendMessage(':new')
    setSessionKey(Date.now().toString())
		navigate('/');
  }, [sendMessage, navigate]);

  const handleChatViewOpen = useCallback(() => {
    navigate('/');
  }, [navigate]);

  const handleHistoryOpen = useCallback(() => {
    listHistory()
    setHistoryDialogOpen(true)
  }, [listHistory]);

  const handleHistoryClose = useCallback(() => {
    setHistoryDialogOpen(false)
  }, []);

  const handleLoadConversation = useCallback((filename: string) => {
    loadConversation(filename)
    setSessionKey(filename)
    handleHistoryClose()
    navigate('/');
  }, [loadConversation, handleHistoryClose, navigate]);

  const handleRenameOpen = useCallback(() => {
    setRenameDialogOpen(true)
  }, []);

  const handleRenameSave = useCallback((newTitle: string) => {
    sendMessage(`:rename ${newTitle}`)
    setRenameDialogOpen(false)
  }, [sendMessage]);

  const handleSourceBrowserOpen = useCallback(() => {
    if (!sourceTree) {
      getSourceTree()
    }
		navigate('/code');
  }, [sourceTree, getSourceTree, navigate]);

  const handleGitBrowserOpen = useCallback(() => {
    getGitGraphLog()
    navigate('/git');
  }, [getGitGraphLog, navigate]);

  const handleReload = useCallback(() => {
    if (view === 'code') {
      getSourceTree();
    } else if (view === 'git') {
      getGitGraphLog();
    }
  }, [view, getSourceTree, getGitGraphLog]);

	const handleToggleLineNumbers = useCallback(() => {
		setShowLineNumbers(prev => !prev)
	}, []);

	const handleModeChange = useCallback((event: SelectChangeEvent) => {
		sendMessage(`:mode ${event.target.value}`)
	}, [sendMessage]);

	const handleModelChange = useCallback((event: SelectChangeEvent) => {
		sendMessage(`:model ${event.target.value}`)
  }, [sendMessage]);

  const handleSendMessage = useCallback((message: string) => {
    sendMessage(message)
  }, [sendMessage]);

  const handleRegenerate = useCallback((index: number) => {
    regenerateFrom(index)
  }, [regenerateFrom]);

  const handleApplyItf = useCallback((content: string) => {
    applyItf(content)
  }, [applyItf]);

  const handleEditMessage = useCallback((index: number, content: string) => {
    editMessage(index, content)
  }, [editMessage]);

  const handleBranchFrom = useCallback((index: number) => {
    branchFrom(index)
  }, [branchFrom]);

  const handleDeleteMessage = useCallback((index: number) => {
    deleteMessage(index)
  }, [deleteMessage]);

  const handleAskAI = useCallback((text: string) => {
    setFloatingChat({ open: true, context: text });
  }, []);

  const handleCloseFloatingChat = useCallback(() => {
    setFloatingChat({ open: false, context: '' });
  }, []);

  const handleFileSelect = useCallback((path: string) => {
    navigate(`/code/${path}`);
  }, [navigate]);

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
					title={view === 'code' ? 'Code' : view === 'git' ? 'Git' : displayedTitle}
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
          onReload={handleReload}
        />
        <Routes>
          <Route path="/code/*" element={
            <SourceBrowser
              tree={sourceTree}
              activeFile={activeFile}
              onFileSelect={handleFileSelect}
              onAskAI={handleAskAI}
              showLineNumbers={showLineNumbers}
            />
          } />
          <Route path="/git/*" element={<GitBrowser log={gitGraphLog} commitDiff={commitDiff} onAskAI={handleAskAI} />} />
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
                // Disable Ask AI in main chat view
                onAskAI={view === 'chat' ? undefined : handleAskAI}
              />
              <ChatInput key={sessionKey} sendMessage={handleSendMessage} cancelGeneration={cancelGeneration} isGenerating={isGenerating} />
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
        currentTitle={finalTitle}
      />
      <FloatingChatWindow
        open={floatingChat.open}
        onClose={handleCloseFloatingChat}
        context={floatingChat.context}
        askAI={askAI}
      />
    </Box>
  )
}

export default App;
