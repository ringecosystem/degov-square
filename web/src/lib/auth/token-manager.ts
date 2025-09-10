'use client';

const TOKEN_KEY = 'degov_auth_token';

class TokenManager {
  private get storage() {
    return typeof window !== 'undefined' ? sessionStorage : null;
  }

  getToken(): string | null {
    if (!this.storage) return null;
    return this.storage.getItem(TOKEN_KEY);
  }

  setToken(token: string | null): void {
    if (!this.storage) return;

    if (token) {
      this.storage.setItem(TOKEN_KEY, token);
    } else {
      this.storage.removeItem(TOKEN_KEY);
    }

    // Dispatch event for auth context to listen to
    window.dispatchEvent(
      new CustomEvent('auth-token-change', {
        detail: { token }
      })
    );
  }

  clearToken(): void {
    this.setToken(null);
  }

  hasValidFormat(): boolean {
    const token = this.getToken();
    return !!(token && token.length > 10);
  }

  getCurrentToken(): string | null {
    return this.getToken();
  }
}

export const tokenManager = new TokenManager();

export const getToken = () => tokenManager.getCurrentToken();
export const setToken = (token: string | null) => tokenManager.setToken(token);
export const clearToken = () => tokenManager.clearToken();
