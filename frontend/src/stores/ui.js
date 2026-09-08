import { defineStore } from 'pinia'

let nextId = 1

/**
 * Transient interface state shared across the app: toasts and the confirmation
 * dialog. Both are rendered once, by App.vue.
 */
export const useUiStore = defineStore('ui', {
  state: () => ({
    toasts: [],
    confirmState: null,
  }),

  actions: {
    notify(message, type = 'info', timeout = 4000) {
      const id = nextId++
      this.toasts.push({ id, message, type })
      if (timeout > 0) {
        setTimeout(() => this.dismiss(id), timeout)
      }
      return id
    },

    success(message) {
      return this.notify(message, 'success')
    },

    error(message) {
      return this.notify(message, 'error', 6000)
    },

    info(message) {
      return this.notify(message, 'info')
    },

    dismiss(id) {
      this.toasts = this.toasts.filter((toast) => toast.id !== id)
    },

    /**
     * Opens the confirmation dialog and resolves to true or false.
     * Usage: if (await ui.confirm({ title: 'Delete project?' })) { ... }
     */
    confirm({ title, message = '', confirmLabel = 'Confirm', cancelLabel = 'Cancel', danger = false }) {
      return new Promise((resolve) => {
        this.confirmState = {
          title,
          message,
          confirmLabel,
          cancelLabel,
          danger,
          resolve,
        }
      })
    },

    resolveConfirm(result) {
      if (this.confirmState) {
        this.confirmState.resolve(result)
        this.confirmState = null
      }
    },
  },
})
