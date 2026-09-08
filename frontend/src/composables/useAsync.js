import { ref, shallowRef } from 'vue'
import { useUiStore } from '@/stores/ui'

/**
 * Wraps an async API call with the loading, error and empty state every list and
 * form in the app needs, so views stop re-implementing the same three refs.
 *
 *   const { data, loading, error, run } = useAsync(() => adminApi.users(query))
 *   onMounted(run)
 */
export function useAsync(fn, { immediate = false, initial = null, toastErrors = false } = {}) {
  const data = shallowRef(initial)
  const meta = ref(null)
  const loading = ref(false)
  const error = ref('')

  async function run(...args) {
    loading.value = true
    error.value = ''
    try {
      const res = await fn(...args)
      // Unwrap the API envelope when present; pass anything else through.
      data.value = res && typeof res === 'object' && 'data' in res ? res.data : res
      meta.value = res?.meta || null
      return data.value
    } catch (err) {
      error.value = err.message || 'Something went wrong'
      if (toastErrors) useUiStore().error(error.value)
      return null
    } finally {
      loading.value = false
    }
  }

  if (immediate) run()

  return { data, meta, loading, error, run }
}

/**
 * Submit helper for forms: tracks the in-flight state and maps field errors from
 * the API onto the form so each input can show its own message.
 */
export function useSubmit(fn, { onSuccess, successMessage } = {}) {
  const submitting = ref(false)
  const error = ref('')
  const fieldErrors = ref({})

  async function submit(...args) {
    submitting.value = true
    error.value = ''
    fieldErrors.value = {}
    try {
      const res = await fn(...args)
      if (successMessage) useUiStore().success(successMessage)
      if (onSuccess) await onSuccess(res)
      return res
    } catch (err) {
      error.value = err.message || 'Something went wrong'
      fieldErrors.value = err.fields || {}
      return null
    } finally {
      submitting.value = false
    }
  }

  return { submit, submitting, error, fieldErrors }
}
