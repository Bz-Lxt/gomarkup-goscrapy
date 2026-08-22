import type { DialogApi, MessageApi } from 'naive-ui'

let messageApi: MessageApi | null = null
let dialogApi: DialogApi | null = null

const toastOpt = { duration: 5000, closable: true, keepAliveOnHover: true }

export function registerFeedback(message: MessageApi, dialog: DialogApi) {
  messageApi = message
  dialogApi = dialog
}

export function toastError(content: string) {
  messageApi?.error(content, toastOpt)
}

export function toastSuccess(content: string) {
  messageApi?.success(content, toastOpt)
}

export function toastInfo(content: string) {
  messageApi?.info(content, toastOpt)
}

export function confirmDanger(opts: {
  title: string
  content: string
  positiveText?: string
}): Promise<boolean> {
  return new Promise((resolve) => {
    if (!dialogApi) {
      resolve(false)
      return
    }
    dialogApi.warning({
      title: opts.title,
      content: opts.content,
      positiveText: opts.positiveText ?? '确认',
      negativeText: '取消',
      closable: true,
      maskClosable: true,
      onPositiveClick: () => resolve(true),
      onNegativeClick: () => resolve(false),
      onClose: () => resolve(false),
    })
  })
}
