export type ViewMode = 'mail' | 'calendar'

class NavigationStore {
  private _currentView = $state<ViewMode>('mail')

  get currentView() {
    return this._currentView
  }

  set currentView(view: ViewMode) {
    this._currentView = view
  }

  setView(view: ViewMode) {
    this._currentView = view
  }
}

export const navigationStore = new NavigationStore()
