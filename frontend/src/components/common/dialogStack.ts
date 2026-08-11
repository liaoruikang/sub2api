let dialogIdCounter = 0
const openDialogStack: number[] = []

export const createDialogId = () => ++dialogIdCounter

export const trackDialogOpen = (dialogId: number) => {
  if (!openDialogStack.includes(dialogId)) openDialogStack.push(dialogId)
}

export const trackDialogClose = (dialogId: number) => {
  const index = openDialogStack.indexOf(dialogId)
  if (index !== -1) openDialogStack.splice(index, 1)
}

export const isTopDialog = (dialogId: number) =>
  openDialogStack[openDialogStack.length - 1] === dialogId

export const hasOpenDialogs = () => openDialogStack.length > 0
