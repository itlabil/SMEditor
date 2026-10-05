import Swal from 'sweetalert2'

// Non-blocking toast notifications, so pages never call alert() directly.
export function useNotify() {
  function success(text) {
    Swal.fire({ icon: 'success', text, toast: true, position: 'top-end', timer: 2500, showConfirmButton: false })
  }

  function error(text) {
    Swal.fire({ icon: 'error', text, toast: true, position: 'top-end', timer: 4000, showConfirmButton: false })
  }

  return { success, error }
}
