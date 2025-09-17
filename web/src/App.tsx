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
	} = useWebSocket(`ws://${location.host}/ws`)
  const [sidebarOpen, setSidebarOpen] = useState(false)
  const [historyDialogOpen, setHistoryDialogOpen] = useState(false)
  const [inputValue, setInputValue] = useState('')

  const handleSidebarToggle = () => {
    setSidebarOpen(!sidebarOpen)
  }

  const handleNewChat = () => {
    sendMessage(':new')
    setInputValue('')
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
        }}
      >
        <TopBar
          onSidebarToggle={handleSidebarToggle}
          title={title}
          tokenCount={tokenCount}
          cwd={cwd}
          mode={mode}
          onModeChange={handleModeChange}
          availableModes={availableModes}
          model={model}
          onModelChange={handleModelChange}
          availableModels={availableModels}
          isGenerating={isGenerating}
        />
        <MessageList
          messages={messages}
          isGenerating={isGenerating}
          onRegenerate={handleRegenerate}
          onApplyItf={handleApplyItf}
          onEditMessage={handleEditMessage}
          onBranchFrom={handleBranchFrom}
          onDeleteMessage={handleDeleteMessage}
        />
        <ChatInput
          sendMessage={handleSendMessage}
          cancelGeneration={cancelGeneration}
          isGenerating={isGenerating}
          value={inputValue}
          onChange={setInputValue}
        />
      </Box>
      <HistoryDialog
        open={historyDialogOpen}
        onClose={handleHistoryClose}
        history={history}
        onLoad={handleLoadConversation}
      />
    </Box>
  )
}

export default App;
