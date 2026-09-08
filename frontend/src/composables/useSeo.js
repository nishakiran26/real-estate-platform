const APP_NAME = import.meta.env.VITE_APP_NAME || 'RERA Property Platform'

/**
 * Writes page metadata into the document head. The API returns a meta title and
 * description for every property, so a listing page is described the way the
 * builder and the platform intend rather than by a generic template.
 *
 * A crawler that does not execute JavaScript still gets the static defaults from
 * index.html plus the server-rendered sitemap; adding SSR later replaces this
 * composable without touching the views.
 */
export function useSeo() {
  function setMeta({ title, description, image, canonical, type = 'website', jsonLd = null }) {
    if (title) {
      document.title = title.includes(APP_NAME) ? title : `${title} | ${APP_NAME}`
    }
    if (description) {
      setTag('name', 'description', description)
      setTag('property', 'og:description', description)
    }
    if (title) {
      setTag('property', 'og:title', title)
    }
    setTag('property', 'og:type', type)
    if (image) {
      setTag('property', 'og:image', image)
      setTag('name', 'twitter:card', 'summary_large_image')
    }

    const url = canonical || window.location.href.split('?')[0]
    setTag('property', 'og:url', url)
    setCanonical(url)
    setJsonLd(jsonLd)
  }

  /**
   * Structured data for a property, so search engines can show price, location
   * and availability in a rich result.
   */
  function propertyJsonLd(property) {
    if (!property) return null
    return {
      '@context': 'https://schema.org',
      '@type': 'Residence',
      name: property.name,
      description: property.meta_description || property.description,
      url: window.location.href,
      image: property.cover_image || undefined,
      address: {
        '@type': 'PostalAddress',
        streetAddress: property.address || undefined,
        addressLocality: property.locality?.name,
        addressRegion: property.city?.name,
        postalCode: property.pincode || undefined,
        addressCountry: 'IN',
      },
      geo:
        property.latitude && property.longitude
          ? {
              '@type': 'GeoCoordinates',
              latitude: property.latitude,
              longitude: property.longitude,
            }
          : undefined,
      numberOfRooms: property.configurations?.[0]?.bedrooms || undefined,
      offers: property.min_price
        ? {
            '@type': 'Offer',
            price: property.min_price,
            priceCurrency: 'INR',
            availability:
              property.construction_status === 'SOLD_OUT'
                ? 'https://schema.org/SoldOut'
                : 'https://schema.org/InStock',
          }
        : undefined,
    }
  }

  return { setMeta, propertyJsonLd }
}

function setTag(attr, key, content) {
  let tag = document.head.querySelector(`meta[${attr}="${key}"]`)
  if (!tag) {
    tag = document.createElement('meta')
    tag.setAttribute(attr, key)
    document.head.appendChild(tag)
  }
  tag.setAttribute('content', content)
}

function setCanonical(url) {
  let link = document.head.querySelector('link[rel="canonical"]')
  if (!link) {
    link = document.createElement('link')
    link.setAttribute('rel', 'canonical')
    document.head.appendChild(link)
  }
  link.setAttribute('href', url)
}

function setJsonLd(data) {
  const existing = document.head.querySelector('script[data-seo-jsonld]')
  if (existing) existing.remove()
  if (!data) return
  const script = document.createElement('script')
  script.type = 'application/ld+json'
  script.setAttribute('data-seo-jsonld', 'true')
  script.textContent = JSON.stringify(data)
  document.head.appendChild(script)
}
