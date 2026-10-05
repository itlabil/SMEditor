import Swal from 'sweetalert2'

// Confirmation dialog for destructive actions (delete project, replace highlight).
export function useConfirm() {
  async function confirm({ title, text, confirmButtonText = 'Ya', cancelButtonText = 'Batal' }) {
    const result = await Swal.fire({
      title,
      text,
      icon: 'warning',
      showCancelButton: true,
      confirmButtonText,
      cancelButtonText,
    })
    return result.isConfirmed
  }

  return { confirm }
}
