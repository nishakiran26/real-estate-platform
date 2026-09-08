import { defineStore } from 'pinia'
import { customerApi, publicApi } from '@/services/api'
import { useAuthStore } from './auth'

const COMPARE_KEY = 'rera_compare'
const MAX_COMPARE = 4

function readCompare() {
  try {
    return JSON.parse(localStorage.getItem(COMPARE_KEY) || '[]')
  } catch {
    return []
  }
}

/**
 * Search results, reference data for the filters, the comparison tray and
 * favourite state. Reference data is fetched once and reused by every filter
 * panel on the site.
 */
export const usePropertyStore = defineStore('property', {
  state: () => ({
    results: [],
    meta: null,
    loading: false,
    error: '',

    reference: null,
    referenceLoading: false,

    compare: readCompare(),
    favoriteIds: new Set(),
  }),

  getters: {
    cities: (state) => state.reference?.cities || [],
    localities: (state) => state.reference?.localities || [],
    amenities: (state) => state.reference?.amenities || [],
    propertyTypes: (state) => state.reference?.property_types || [],
    constructionStatuses: (state) => state.reference?.construction_status || [],
    budgetRanges: (state) => state.reference?.budget_ranges || [],
    sortOptions: (state) => state.reference?.sort_options || [],

    // Rental vocabulary, served by /meta so it can change without a deploy.
    rentRanges: (state) => state.reference?.rent_ranges || [],
    furnishingOptions: (state) => state.reference?.furnishing || [],
    tenantPreferences: (state) => state.reference?.tenant_preference || [],
    availabilityOptions: (state) => state.reference?.availability || [],
    rentalPhotoTypes: (state) => state.reference?.rental_photo_types || [],
    includedItemOptions: (state) => state.reference?.included_items || [],
    compareCount: (state) => state.compare.length,
    compareIds: (state) => state.compare.map((item) => item.id),
    canCompareMore: (state) => state.compare.length < MAX_COMPARE,
  },

  actions: {
    /** Loads cities, localities, amenities and filter enumerations once. */
    async loadReference(force = false) {
      if (this.reference && !force) return this.reference
      if (this.referenceLoading) return this.reference
      this.referenceLoading = true
      try {
        const res = await publicApi.meta()
        this.reference = res.data
        return this.reference
      } catch {
        return null
      } finally {
        this.referenceLoading = false
      }
    },

    localitiesForCity(citySlug) {
      if (!citySlug) return this.localities
      const city = this.cities.find((item) => item.slug === citySlug)
      if (!city) return []
      return this.localities.filter((locality) => locality.city_id === city.id)
    },

    async search(params) {
      this.loading = true
      this.error = ''
      try {
        const res = await publicApi.properties(params)
        this.results = res.data || []
        this.meta = res.meta || null
        this.syncFavoritesFromCards(this.results)
        return this.results
      } catch (err) {
        this.error = err.message
        this.results = []
        this.meta = null
        return []
      } finally {
        this.loading = false
      }
    },

    syncFavoritesFromCards(cards) {
      for (const card of cards) {
        if (card.is_favorite) this.favoriteIds.add(card.id)
      }
    },

    isFavorite(propertyId) {
      return this.favoriteIds.has(propertyId)
    },

    /**
     * Toggles a saved property. Returns 'added', 'removed', or 'unauthenticated'
     * so the caller can prompt for a login without this store knowing about the
     * router.
     */
    async toggleFavorite(propertyId) {
      const auth = useAuthStore()
      if (!auth.isAuthenticated) return 'unauthenticated'

      if (this.favoriteIds.has(propertyId)) {
        this.favoriteIds.delete(propertyId)
        try {
          await customerApi.removeFavorite(propertyId)
          return 'removed'
        } catch (err) {
          this.favoriteIds.add(propertyId) // roll back the optimistic update
          throw err
        }
      }

      this.favoriteIds.add(propertyId)
      try {
        await customerApi.addFavorite(propertyId)
        return 'added'
      } catch (err) {
        this.favoriteIds.delete(propertyId)
        throw err
      }
    },

    async loadFavorites() {
      const auth = useAuthStore()
      if (!auth.isAuthenticated) return []
      try {
        const res = await customerApi.favorites({ per_page: 100 })
        this.favoriteIds = new Set((res.data || []).map((card) => card.id))
        return res.data || []
      } catch {
        return []
      }
    },

    // --- Comparison tray ---------------------------------------------------

    inCompare(propertyId) {
      return this.compare.some((item) => item.id === propertyId)
    },

    toggleCompare(card) {
      if (this.inCompare(card.id)) {
        this.compare = this.compare.filter((item) => item.id !== card.id)
      } else {
        if (this.compare.length >= MAX_COMPARE) return false
        this.compare = [...this.compare, { id: card.id, name: card.name, slug: card.slug }]
      }
      this.persistCompare()
      return true
    },

    clearCompare() {
      this.compare = []
      this.persistCompare()
    },

    persistCompare() {
      localStorage.setItem(COMPARE_KEY, JSON.stringify(this.compare))
    },
  },
})

export { MAX_COMPARE }
