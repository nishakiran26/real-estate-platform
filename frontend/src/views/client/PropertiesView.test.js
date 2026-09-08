import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import PropertiesView from './PropertiesView.vue'

const setAvailability = vi.fn()
const renewListing = vi.fn()
const expiringListings = vi.fn()
const properties = vi.fn()

vi.mock('@/services/api', () => ({
  clientApi: {
    properties: (...args) => properties(...args),
    setAvailability: (...args) => setAvailability(...args),
    renewListing: (...args) => renewListing(...args),
    expiringListings: (...args) => expiringListings(...args),
    deleteProperty: vi.fn(),
    submitProperty: vi.fn(),
  },
}))
vi.mock('@/services/http', () => ({ configureAuth: vi.fn() }))

const rental = {
  id: 1,
  name: 'Wakad 2 BHK',
  purpose: 'RENT',
  property_status: 'PUBLISHED',
  availability_status: 'AVAILABLE',
  view_count: 40,
  created_at: '2026-09-01T10:00:00Z',
}

const sale = {
  id: 2,
  name: 'Green Valley Residency',
  purpose: 'BUY',
  property_status: 'PUBLISHED',
  availability_status: 'AVAILABLE',
  view_count: 90,
  created_at: '2026-09-01T10:00:00Z',
}

function render(items = [rental, sale], expiring = []) {
  // The view marks a row rented in place, so it mutates the objects it is
  // given. Handing every test its own copies keeps one test's change out of
  // the next one's fixture.
  properties.mockImplementation(() =>
    Promise.resolve({
      data: items.map((item) => ({ ...item })),
      meta: { page: 1, total_pages: 1 },
    }),
  )
  expiringListings.mockResolvedValue({ data: expiring })
  return mount(PropertiesView, {
    global: { stubs: { RouterLink: { template: '<a><slot /></a>' } } },
  })
}

describe('the owner property table', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('offers availability on a rental and nothing on a flat for sale', async () => {
    const wrapper = render()
    await flushPromises()

    // One select, on the rental row only: a flat for sale has no availability
    // to manage, and offering the control would imply otherwise.
    const selects = wrapper.findAll('tbody select')
    expect(selects).toHaveLength(1)
    expect(selects[0].element.value).toBe('AVAILABLE')
    expect(wrapper.findAll('tbody tr')).toHaveLength(2)
  })

  it('marks a flat rented without sending the listing back for review', async () => {
    const wrapper = render()
    await flushPromises()
    setAvailability.mockResolvedValue({ data: {} })

    const select = wrapper.find('tbody select')
    await select.setValue('RENTED')
    await flushPromises()

    expect(setAvailability).toHaveBeenCalledWith(1, { availability_status: 'RENTED' })
    // Availability is not the approval workflow, so nothing resubmits the
    // listing and the table is not reloaded behind the owner's back.
    expect(properties).toHaveBeenCalledTimes(1)
  })

  it('puts the row back as it was when the server refuses', async () => {
    const wrapper = render()
    await flushPromises()
    setAvailability.mockRejectedValue(new Error('Availability applies to rental listings only'))

    await wrapper.find('tbody select').setValue('RENTED')
    await flushPromises()

    // A failed change must not leave the screen claiming the flat is let, so
    // the table is reloaded from the server.
    expect(properties).toHaveBeenCalledTimes(2)
  })

  it('warns about listings that lapse soon and renews one in place', async () => {
    const wrapper = render(
      [rental],
      [{ id: 1, name: 'Wakad 2 BHK', days_left: 2, has_expired: false }],
    )
    await flushPromises()

    expect(wrapper.text()).toContain('1 rental listing(s) lapse soon')
    expect(wrapper.text()).toContain('2 day(s) left')

    renewListing.mockResolvedValue({ data: {} })
    await wrapper.find('.expiry-list button').trigger('click')
    await flushPromises()

    expect(renewListing).toHaveBeenCalledWith(1)
  })

  it('stays silent when nothing is about to lapse', async () => {
    const wrapper = render()
    await flushPromises()

    expect(wrapper.text()).not.toContain('lapse soon')
  })
})
