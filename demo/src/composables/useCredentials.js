import { ref, reactive } from 'vue'
import { useBridge } from './useBridge.js'

export function useCredentials() {
  const { bridge } = useBridge()

  const credList = ref([])
  const credForm = reactive({ username: '', password: '' })

  async function loadCreds() {
    try {
      const result = await bridge.getCredentials()
      credList.value = result.credentials || []
    } catch {}
  }

  async function saveCred() {
    if (!credForm.username || !credForm.password) return
    try {
      await bridge.saveCredentials({ username: credForm.username, password: credForm.password })
      credForm.password = ''
      await loadCreds()
    } catch (err) { alert('保存失败: ' + (err?.message || err)) }
  }

  async function deleteCred(username) {
    try {
      await bridge.deleteCredentials({ username })
      await loadCreds()
    } catch (err) { alert('删除失败: ' + (err?.message || err)) }
  }

  async function clearCreds() {
    if (!confirm('确定清除所有已保存的凭据？')) return
    try {
      await bridge.clearCredentials()
      await loadCreds()
    } catch (err) { alert('清除失败: ' + (err?.message || err)) }
  }

  return { credList, credForm, loadCreds, saveCred, deleteCred, clearCreds }
}
