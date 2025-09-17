import { useState } from 'react';
import {
  Box,
  CssBaseline,
	type SelectChangeEvent,
} from '@mui/material'
import { useWebSocket } from './hooks/useWebSocket'
import { Sidebar } from './components/Sidebar'
import { MessageList } from './components/MessageList'
import { ChatInput } from './components/ChatInput'
import { TopBar } from './components/TopBar';
import { HistoryDialog } from './components/HistoryDialog';
import { RenameDialog } from './components/RenameDialog';
import { SourceBrowser } from './components/SourceBrowser';

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
	} = useWebSocket(`ws://${location.host}/ws`)
	const [sidebarOpen, setSidebarOpen] = useState(false)
	const [historyDialogOpen, setHistoryDialogOpen] = useState(false)
	const [renameDialogOpen, setRenameDialogOpen] = useState(false)
	const [inputValue, setInputValue] = useState('')
	const [view, setView] = useState<'chat' | 'code'>('chat')
	const [showLineNumbers, setShowLineNumbers] = useState(false)

  const handleSidebarToggle = () => {
    setSidebarOpen(!sidebarOpen)
  }

  const handleNewChat = () => {
    sendMessage(':new')
    setInputValue('')
		setView('chat')
  }

  const handleChatViewOpen = () => {
    setView('chat')
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
    setView('chat')
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
		setView('code')
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
					title={view === 'code' ? 'Code' : title}
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
				{view === 'chat' ? (
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
				) : (
					<SourceBrowser
						tree={sourceTree}
						activeFile={activeFile}
						onFileSelect={getFileContent}
						showLineNumbers={showLineNumbers}
						cwd={cwd}
					/>
				)}
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
