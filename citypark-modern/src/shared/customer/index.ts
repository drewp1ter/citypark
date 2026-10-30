export { customerRequest, CustomerApiError, customerFallback } from './customer'
export { clearCustomerSession, getCustomerSessionToken, hasCustomerSessionCookie, setCustomerSession, getGuestCheckoutCookie, saveGuestCheckoutCookie, addGoogleAttempt, consumeGoogleAttempt } from './customer-session'
export { customerAuthorized } from './customer'
export type { CustomerProfile, CustomerMethod, CustomerPairing } from './customer'
