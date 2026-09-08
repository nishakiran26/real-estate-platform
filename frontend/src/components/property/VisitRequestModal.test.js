import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import VisitRequestModal from './VisitRequestModal.vue'
import { useAuthStore } from '@/stores/auth'

const requestVisit = vi.fn()

vi.mock('@/services/api', () => ({
  authApi: {},
  publicApi: { requestVisit: (...args) => requestVisit(...args) },
}))
vi.mock('@/services/http', () => ({ configureAuth: vi.fn() }))

const property = { id: 7, name: 'Wakad 2 BHK', slug: 'wakad-2-bhk-wakad-pune' }

function render() {
  return mount(VisitRequestModal, { props: { property } })
}

// Fill the fields a visitor must supply, leaving consent alone so each test can
// decide whether it was given.
async function fillRequiredFields(wrapper) {
  const inputs = wrapper.findAll('input.input')
  await wrapper.find('input[type="date"]').setValue('2026-10-01')
  await inputs.find((i) => i.attributes('type') === undefined).setValue('Anjali Rao')
  await wrapper.find('input[inputmode="numeric"]').setValue('9812340001')
}

describe('VisitRequestModal', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    localStorage.clear()
    vi.clearAllMocks()
  })

  it('does not make a signed-in visitor retype what the platform knows', () => {
    const auth = useAuthStore()
    auth.user = { full_name: 'Anjali Rao', mobile: '9812340001', email: 'anjali@example.com' }

    const wrapper = render()

    expect(wrapper.find('input[inputmode="numeric"]').element.value).toBe('9812340001')
    expect(wrapper.find('input[type="email"]').element.value).toBe('anjali@example.com')
  })

  it('sends the request against the property being viewed', async () => {
    requestVisit.mockResolvedValue({ data: { status: 'REQUESTED' } })
    const wrapper = render()
    await fillRequiredFields(wrapper)
    await wrapper.find('input[type="checkbox"]').setValue(true)

    await wrapper.find('form').trigger('submit.prevent')
    await flushPromises()

    expect(requestVisit).toHaveBeenCalledTimes(1)
    const [slug, payload] = requestVisit.mock.calls[0]
    expect(slug).toBe(property.slug)
    expect(payload).toMatchObject({
      name: 'Anjali Rao',
      mobile: '9812340001',
      visit_date: '2026-10-01',
      time_slot: 'EVENING',
      consent_contact: true,
    })
  })

  it('confirms rather than leaving the form on screen to be sent twice', async () => {
    requestVisit.mockResolvedValue({ data: { status: 'REQUESTED' } })
    const wrapper = render()
    await fillRequiredFields(wrapper)
    await wrapper.find('input[type="checkbox"]').setValue(true)

    await wrapper.find('form').trigger('submit.prevent')
    await flushPromises()

    expect(wrapper.text()).toContain('Request sent')
    expect(wrapper.find('form').exists()).toBe(false)
  })

  it('keeps the form and shows why when the server refuses', async () => {
    requestVisit.mockRejectedValue(
      Object.assign(new Error('Pick a date in the future'), {
        fields: { visit_date: 'Pick a date in the future' },
      }),
    )
    const wrapper = render()
    await fillRequiredFields(wrapper)
    await wrapper.find('input[type="checkbox"]').setValue(true)

    await wrapper.find('form').trigger('submit.prevent')
    await flushPromises()

    expect(wrapper.find('form').exists()).toBe(true)
    expect(wrapper.text()).toContain('Pick a date in the future')
    expect(wrapper.find('.field-error').exists()).toBe(true)
  })

  it('cannot be sent without consent to be contacted', () => {
    const consent = render().find('input[type="checkbox"]')

    // Consent is required by the API, so the form must require it too rather
    // than letting the visitor submit and be rejected.
    expect(consent.attributes('required')).toBeDefined()
  })

  it('closes on the backdrop and on Cancel', async () => {
    const wrapper = render()

    await wrapper.find('.modal-backdrop').trigger('click')
    await wrapper.find('.modal-actions .btn-outline').trigger('click')

    expect(wrapper.emitted('close')).toHaveLength(2)
  })
})
