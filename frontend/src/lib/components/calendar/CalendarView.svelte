<script lang="ts">
  import Icon from '@iconify/svelte'
  import { onMount } from 'svelte'
  import { _ } from '$lib/i18n'
  import { Button } from '$lib/components/ui/button'
  import * as AlertDialog from '$lib/components/ui/alert-dialog'
  import { navigationStore } from '$lib/stores/navigation.svelte'
  import { accountStore } from '$lib/stores/accounts.svelte'
  import EventDetailsDialog from './EventDetailsDialog.svelte'
  import DayView from './DayView.svelte'
  import EventDialog from './EventDialog.svelte'
  // @ts-ignore - wailsjs
  import { GetCalendars, GetCalendarEvents, SyncCalendars, DeleteCalendarEvent, SetCalendarEnabled, SetCalendarsEnabled } from '../../../../wailsjs/go/app/App'
  // @ts-ignore - wailsjs models
  import { calendar } from '../../../../wailsjs/go/models'
  import { format, startOfMonth, endOfMonth, startOfWeek, endOfWeek, eachDayOfInterval, isSameMonth, isSameDay, addMonths, subMonths } from 'date-fns'

  let calendars = $state<calendar.Calendar[]>([])
  let events = $state<any[]>([])
  let currentMonth = $state(new Date())
  let loading = $state(false)
  let enabledCalendars = $state<Record<string, boolean>>({})
  let expandedSections = $state<Record<string, boolean>>({ 'shared': true, 'accounts_root': true })

  // View state
  let viewMode = $state<'month' | 'day'>('month')
  let selectedDate = $state<Date>(new Date())

  // Dialog state
  let showDetails = $state(false)
  let selectedEvent = $state<calendar.Event | null>(null)
  let showDeleteConfirm = $state(false)
  let showEventDialog = $state(false)
  let eventDialogInitialDate = $state<Date | undefined>(undefined)

  // Group calendars by account for the sidebar
  const groupedCalendars = $derived(() => {
    const groups: Record<string, { account: any, calendars: calendar.Calendar[] }> = {}
    for (const cal of calendars) {
      if (!groups[cal.accountId]) {
        const acc = accountStore.accounts.find(a => a.account.id === cal.accountId)
        groups[cal.accountId] = { account: acc?.account, calendars: [] }
      }
      groups[cal.accountId].calendars.push(cal)
    }
    return groups
  })

  // Group shared/global calendars for the top section
  const sharedCalendars = $derived(() => {
    const shared: Record<string, { name: string, instances: calendar.Calendar[] }> = {}
    for (const cal of calendars) {
      if (cal.isGlobal && cal.canonicalId) {
        if (!shared[cal.canonicalId]) {
          shared[cal.canonicalId] = { name: cal.name, instances: [] }
        }
        shared[cal.canonicalId].instances.push(cal)
      }
    }
    return shared
  })

  async function toggleCalendar(calId: string) {
    const newState = !enabledCalendars[calId]
    enabledCalendars[calId] = newState
    try {
      await SetCalendarEnabled(calId, newState)
      await loadData()
    } catch (err) {
      console.error('Failed to update calendar status:', err)
    }
  }

  async function toggleSharedCalendar(canonicalId: string, instances: calendar.Calendar[]) {
    const newState = !instances.every(inst => enabledCalendars[inst.id])
    const ids = instances.map(inst => inst.id)
    for (const id of ids) {
      enabledCalendars[id] = newState
    }
    try {
      await SetCalendarsEnabled(ids, newState)
      await loadData()
    } catch (err) {
      console.error('Failed to update shared calendars status:', err)
    }
  }

  async function toggleAllSharedCalendars() {
    const allShared = Object.values(sharedCalendars()).flatMap(s => s.instances)
    const newState = !allShared.every(inst => enabledCalendars[inst.id])
    const ids = allShared.map(inst => inst.id)
    for (const id of ids) {
      enabledCalendars[id] = newState
    }
    try {
      await SetCalendarsEnabled(ids, newState)
      await loadData()
    } catch (err) {
      console.error('Failed to update all shared calendars status:', err)
    }
  }

  async function toggleAccountCalendars(calendars: calendar.Calendar[]) {
    const newState = !calendars.every(inst => enabledCalendars[inst.id])
    const ids = calendars.map(inst => inst.id)
    for (const id of ids) {
      enabledCalendars[id] = newState
    }
    try {
      await SetCalendarsEnabled(ids, newState)
      await loadData()
    } catch (err) {
      console.error('Failed to update account calendars status:', err)
    }
  }

  function toggleSection(sectionId: string) {
    expandedSections[sectionId] = !expandedSections[sectionId]
  }

  function openEvent(event: calendar.Event) {
    selectedEvent = event
    showDetails = true
  }

  function requestDeleteEvent() {
    showDeleteConfirm = true
  }

  async function confirmDeleteEvent() {
    if (!selectedEvent) return
    try {
      await DeleteCalendarEvent(selectedEvent.calendarId, selectedEvent.id)
      showDetails = false
      showDeleteConfirm = false
      await loadData()
    } catch (err) {
      console.error('Failed to delete event:', err)
    }
  }

  function openNewEvent(date?: Date) {
    eventDialogInitialDate = date
    showEventDialog = true
  }

  function openDayView(day: Date) {
    selectedDate = day
    viewMode = 'day'
  }

  function closeDayView() {
    viewMode = 'month'
  }

  async function loadData() {
    loading = true
    try {
      calendars = await GetCalendars()
      
      // Initialize enabled state for new calendars
      for (const cal of calendars) {
        if (enabledCalendars[cal.id] === undefined) {
          enabledCalendars[cal.id] = cal.enabled
        }
        if (expandedSections[cal.accountId] === undefined) {
          expandedSections[cal.accountId] = true
        }
      }
      
      const start = startOfWeek(startOfMonth(currentMonth))
      const end = endOfWeek(endOfMonth(currentMonth))
      
      const eventsMap = new Map<string, any>()

      for (const cal of calendars) {
        if (enabledCalendars[cal.id]) {
          const calEvents = await GetCalendarEvents(cal.id, start.toISOString(), end.toISOString())
          if (calEvents) {
            for (const e of calEvents) {
              const key = e.remoteId || e.id
              const acc = accountStore.accounts.find(a => a.account.id === cal.accountId)
              const color = acc?.account?.color || '#ccc'

              if (!eventsMap.has(key)) {
                eventsMap.set(key, { ...e, accountColors: [color] })
              } else {
                const existing = eventsMap.get(key)
                if (!existing.accountColors.includes(color)) {
                  existing.accountColors.push(color)
                }
              }
            }
          }
        }
      }
      events = Array.from(eventsMap.values())
    } catch (err) {
      console.error('Failed to load calendar data:', err)
    } finally {
      loading = false
    }
  }

  onMount(() => {
    loadData()
  })

  function nextMonth() {
    currentMonth = addMonths(currentMonth, 1)
    loadData()
  }

  function prevMonth() {
    currentMonth = subMonths(currentMonth, 1)
    loadData()
  }

  function goToToday() {
    currentMonth = new Date()
    loadData()
  }

  async function handleSync() {
    loading = true
    try {
      await SyncCalendars()
      // Wait a bit for background sync to do some work
      setTimeout(async () => {
        await loadData()
        loading = false
      }, 3000)
    } catch (err) {
      console.error('Manual sync failed:', err)
      loading = false
    }
  }

  const days = $derived(eachDayOfInterval({
    start: startOfWeek(startOfMonth(currentMonth)),
    end: endOfWeek(endOfMonth(currentMonth))
  }))

  function getEventsForDay(day: Date) {
    return events.filter(e => isSameDay(new Date(e.startTime), day))
  }
</script>

<div class="flex flex-col h-full bg-background">
  <!-- Calendar Header -->
  <header class="flex items-center justify-between px-6 py-4 border-b border-border">
    <div class="flex items-center gap-4">
      <h1 class="text-xl font-semibold">{format(currentMonth, 'MMMM yyyy')}</h1>
      <div class="flex items-center bg-muted rounded-md p-1">
        <button class="p-1 hover:bg-background rounded transition-colors" onclick={prevMonth}>
          <Icon icon="mdi:chevron-left" class="w-5 h-5" />
        </button>
        <button class="px-3 py-1 text-sm font-medium hover:bg-background rounded transition-colors" onclick={goToToday}>
          {$_('calendar.today')}
        </button>
        <button class="p-1 hover:bg-background rounded transition-colors" onclick={nextMonth}>
          <Icon icon="mdi:chevron-right" class="w-5 h-5" />
        </button>
      </div>
    </div>
    
    <div class="flex items-center gap-2">
      <Button variant="outline" size="sm" onclick={() => openNewEvent()} disabled={loading}>
        <Icon icon="mdi:plus" class="w-4 h-4 mr-2" />
        {$_('calendar.newEvent') || 'New Event'}
      </Button>
      <Button variant="outline" size="sm" onclick={handleSync} disabled={loading}>
        <Icon icon="mdi:sync" class="w-4 h-4 mr-2 {loading ? 'animate-spin' : ''}" />
        {$_('common.sync')}
      </Button>
      <Button size="sm" onclick={() => navigationStore.setView('mail')}>
        <Icon icon="mdi:email" class="w-4 h-4 mr-2" />
        {$_('common.back')}
      </Button>
    </div>
  </header>

  <!-- Main Calendar View with Sidebar -->
  <div class="flex-1 flex overflow-hidden">
    <!-- Calendar Sidebar (List of calendars) -->
    <aside class="w-64 border-r border-border bg-muted/20 overflow-y-auto p-4 flex flex-col gap-6">
      {#if Object.keys(sharedCalendars()).length > 0}
        <div class="space-y-2">
          <button 
            class="w-full flex items-center justify-between px-2 py-1 group"
            onclick={() => toggleSection('shared')}
          >
            <h2 class="text-xs font-bold text-muted-foreground uppercase tracking-wider">
              {$_('calendar.shared') || 'Sincronizados'}
            </h2>
            <Icon 
              icon="mdi:chevron-down" 
              class="w-4 h-4 text-muted-foreground transition-transform {expandedSections['shared'] ? '' : '-rotate-90'}" 
            />
          </button>
          
          {#if expandedSections['shared']}
            <div class="space-y-4">
              {#each Object.entries(sharedCalendars()) as [canonicalId, shared]}
                <div class="space-y-1">
                  <div class="px-2 py-1 flex items-center justify-between group">
                    <button 
                      class="flex-1 flex items-center gap-2 text-left"
                      onclick={() => toggleSection(canonicalId)}
                    >
                      <span class="text-[10px] font-bold text-muted-foreground uppercase tracking-wider flex-1">{shared.name}</span>
                      <Icon 
                        icon="mdi:chevron-down" 
                        class="w-3 h-3 text-muted-foreground transition-transform {expandedSections[canonicalId] !== false ? '' : '-rotate-90'}" 
                      />
                    </button>
                    <button
                      class="ml-2 p-1 rounded hover:bg-background/80 transition-colors"
                      onclick={(e) => { e.stopPropagation(); toggleSharedCalendar(canonicalId, shared.instances) }}
                    >
                      <div 
                        class="w-3 h-3 rounded-sm border flex items-center justify-center transition-colors"
                        style="background-color: {shared.instances.every(inst => enabledCalendars[inst.id]) ? (shared.instances[0]?.color || 'var(--primary)') : 'transparent'}; border-color: {shared.instances[0]?.color || 'var(--primary)'}"
                      >
                        {#if shared.instances.every(inst => enabledCalendars[inst.id])}
                          <Icon icon="mdi:check" class="w-2.5 h-2.5 text-white" />
                        {/if}
                      </div>
                    </button>
                  </div>
                  {#if expandedSections[canonicalId] !== false}
                    <div class="space-y-1">
                      {#each shared.instances as inst}
                        {@const acc = accountStore.accounts.find(a => a.account.id === inst.accountId)}
                        <button
                          class="w-full flex items-center gap-3 px-3 py-2 rounded-md transition-colors hover:bg-background/80 group"
                          onclick={() => toggleCalendar(inst.id)}
                        >
                          <div 
                            class="w-4 h-4 rounded border flex items-center justify-center transition-colors"
                            style="background-color: {enabledCalendars[inst.id] ? inst.color : 'transparent'}; border-color: {inst.color}"
                          >
                            {#if enabledCalendars[inst.id]}
                              <Icon icon="mdi:check" class="w-3 h-3 text-white" />
                            {/if}
                          </div>
                          <div class="flex-1 min-w-0 flex items-center gap-2">
                            <div 
                              class="w-2 h-2 rounded-full flex-shrink-0" 
                              style="background-color: {acc?.account?.color || '#ccc'}"
                            ></div>
                            <span class="text-sm truncate text-left {enabledCalendars[inst.id] ? 'text-foreground font-medium' : 'text-muted-foreground'}">
                              {acc?.account?.name || inst.accountId}
                            </span>
                          </div>
                        </button>
                      {/each}
                    </div>
                  {/if}
                </div>
              {/each}
            </div>
          {/if}
        </div>
        <div class="border-b border-border mx-3 my-1"></div>
      {/if}

      <div class="space-y-2">
        <button 
          class="w-full flex items-center justify-between px-2 py-1 group"
          onclick={() => toggleSection('accounts_root')}
        >
          <h2 class="text-xs font-bold text-muted-foreground uppercase tracking-wider">
            {$_('settings.accounts')}
          </h2>
          <Icon 
            icon="mdi:chevron-down" 
            class="w-4 h-4 text-muted-foreground transition-transform {expandedSections['accounts_root'] ? '' : '-rotate-90'}" 
          />
        </button>
        
        {#if expandedSections['accounts_root']}
          <div class="space-y-4">
            {#each Object.entries(groupedCalendars()) as [accId, group]}
              <div class="space-y-2">
                <div class="flex items-center justify-between px-2 py-1 group">
                  <button 
                    class="flex-1 flex items-center gap-2"
                    onclick={() => toggleSection(accId)}
                  >
                    <div class="w-2 h-2 rounded-full" style="background-color: {group.account?.color || '#ccc'}"></div>
                    <span class="text-xs font-semibold truncate text-muted-foreground flex-1 text-left">{group.account?.name || accId}</span>
                    <Icon 
                      icon="mdi:chevron-down" 
                      class="w-3 h-3 text-muted-foreground transition-transform {expandedSections[accId] ? '' : '-rotate-90'}" 
                    />
                  </button>
                  <button
                    class="ml-2 p-1 rounded hover:bg-background/80 transition-colors"
                    onclick={(e) => { e.stopPropagation(); toggleAccountCalendars(group.calendars) }}
                  >
                    <div 
                      class="w-3 h-3 rounded-sm border flex items-center justify-center transition-colors"
                      style="background-color: {group.calendars.every(inst => enabledCalendars[inst.id]) ? (group.account?.color || 'var(--primary)') : 'transparent'}; border-color: {group.account?.color || 'var(--primary)'}"
                    >
                      {#if group.calendars.every(inst => enabledCalendars[inst.id])}
                        <Icon icon="mdi:check" class="w-2 h-2 text-white" />
                      {/if}
                    </div>
                  </button>
                </div>
                
                {#if expandedSections[accId]}
                  <div class="space-y-1">
                    {#each group.calendars as cal}
                      <button
                        class="w-full flex items-center gap-3 px-3 py-2 rounded-md transition-colors hover:bg-background/80 group"
                        onclick={() => toggleCalendar(cal.id)}
                      >
                        <div 
                          class="w-4 h-4 rounded border flex items-center justify-center transition-colors"
                          style="background-color: {enabledCalendars[cal.id] ? cal.color : 'transparent'}; border-color: {cal.color}"
                        >
                          {#if enabledCalendars[cal.id]}
                            <Icon icon="mdi:check" class="w-3 h-3 text-white" />
                          {/if}
                        </div>
                        <span class="text-sm truncate flex-1 text-left {enabledCalendars[cal.id] ? 'text-foreground font-medium' : 'text-muted-foreground'}">
                          {cal.name}
                        </span>
                      </button>
                    {/each}
                  </div>
                {/if}
              </div>
            {/each}
          </div>
        {/if}
      </div>
    </aside>

    <!-- Calendar Grid Area -->
    <div class="flex-1 flex flex-col relative overflow-hidden">
      {#if !loading && calendars.length === 0}
        <div class="absolute inset-0 z-10 flex flex-col items-center justify-center bg-background/80 backdrop-blur-sm p-4 text-center">
          <Icon icon="mdi:calendar-alert" class="w-16 h-16 text-muted-foreground mb-4" />
          <h2 class="text-xl font-semibold mb-2">{$_('calendar.noCalendarsFound')}</h2>
          <p class="text-muted-foreground max-w-md mb-6">
            To see your Google Calendar events, you must connect your account using <strong>OAuth2</strong>. App Passwords are not supported for the Calendar API.
          </p>
          <Button onclick={() => navigationStore.setView('mail')}>
            Go to Accounts
          </Button>
        </div>
      {:else if !loading && Object.values(enabledCalendars).every(v => !v)}
         <div class="absolute inset-0 z-10 flex flex-col items-center justify-center bg-background/80 backdrop-blur-sm p-4 text-center">
          <Icon icon="mdi:calendar-blank" class="w-16 h-16 text-muted-foreground mb-4" />
          <h2 class="text-xl font-semibold mb-2">No calendars selected</h2>
          <p class="text-muted-foreground max-w-md mb-6">
            Select one or more calendars from the sidebar to view your events.
          </p>
        </div>
      {/if}
      
      {#if viewMode === 'month'}
        <!-- Day Names -->
        <div class="grid grid-cols-7 border-b border-border bg-muted/30">
          {#each ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'] as day}
            <div class="py-2 text-center text-xs font-semibold text-muted-foreground uppercase tracking-wider">
              {day}
            </div>
          {/each}
        </div>

        <!-- Days Grid -->
        <div class="flex-1 grid grid-cols-7 grid-rows-6 overflow-auto bg-background">
          {#each days as day}
            <div 
              class="min-h-[100px] border-r border-b border-border p-2 cursor-pointer hover:bg-muted/5 transition-colors {isSameMonth(day, currentMonth) ? '' : 'bg-muted/10'} group/day"
              onclick={() => openDayView(day)}
            >
              <div class="flex justify-between items-start mb-1">
                <span class="text-sm font-medium {isSameDay(day, new Date()) ? 'bg-primary text-primary-foreground w-6 h-6 flex items-center justify-center rounded-full' : 'text-muted-foreground'}">
                  {format(day, 'd')}
                </span>
                <button 
                  class="p-1 rounded hover:bg-primary/10 text-primary opacity-0 group-hover/day:opacity-100 transition-opacity"
                  onclick={(e) => { e.stopPropagation(); openNewEvent(day) }}
                  title="New Event"
                >
                  <Icon icon="mdi:plus" class="w-4 h-4" />
                </button>
              </div>
              
              <div class="space-y-1">
                {#each getEventsForDay(day) as event}
                  <button 
                    class="w-full text-left px-2 py-0.5 text-[10px] sm:text-xs rounded border truncate hover:brightness-95 transition-all flex items-center gap-1"
                    style="background-color: {calendars.find(c => c.id === event.calendarId)?.color || '#3b82f6'}22; border-color: {calendars.find(c => c.id === event.calendarId)?.color || '#3b82f6'}"
                    onclick={(e) => { e.stopPropagation(); openEvent(event) }}
                    title={event.summary}
                  >
                    {#if !event.isAllDay}
                      <span class="font-bold opacity-70 whitespace-nowrap">{format(new Date(event.startTime), 'HH:mm')}</span>
                    {/if}
                    <span class="truncate flex-1">{event.summary || $_('calendar.noSubject')}</span>
                    
                    {#if event.accountColors && event.accountColors.length > 1}
                      <div class="flex -space-x-1 ml-1">
                        {#each event.accountColors as color}
                          <div class="w-1.5 h-1.5 rounded-full border-[0.5px] border-background" style="background-color: {color}"></div>
                        {/each}
                      </div>
                    {/if}

                    {#if event.meetLink}
                      <Icon icon="mdi:video" class="w-3 h-3 ml-auto opacity-70" />
                    {/if}
                  </button>
                {/each}
              </div>
            </div>
          {/each}
        </div>
      {:else}
        <DayView 
          date={selectedDate} 
          events={events} 
          calendars={calendars} 
          onClose={closeDayView}
          onOpenEvent={openEvent}
          onRefresh={loadData}
        />
      {/if}
    </div>
  </div>
</div>

<EventDetailsDialog
  bind:open={showDetails}
  event={selectedEvent}
  calendarColor={calendars.find(c => c.id === selectedEvent?.calendarId)?.color}
  onDelete={requestDeleteEvent}
/>

<EventDialog
  bind:open={showEventDialog}
  initialDate={eventDialogInitialDate}
  calendars={calendars}
  onSave={loadData}
/>

<AlertDialog.Root bind:open={showDeleteConfirm}>
  <AlertDialog.Content>
    <AlertDialog.Header>
      <AlertDialog.Title>{$_('calendar.deleteEvent')}</AlertDialog.Title>
      <AlertDialog.Description>
        {$_('dialog.deleteDescription')}
      </AlertDialog.Description>
    </AlertDialog.Header>
    <AlertDialog.Footer>
      <AlertDialog.Cancel>{$_('common.cancel')}</AlertDialog.Cancel>
      <AlertDialog.Action onclick={confirmDeleteEvent} class="bg-destructive text-destructive-foreground hover:bg-destructive/90">
        {$_('common.delete')}
      </AlertDialog.Action>
    </AlertDialog.Footer>
  </AlertDialog.Content>
</AlertDialog.Root>
