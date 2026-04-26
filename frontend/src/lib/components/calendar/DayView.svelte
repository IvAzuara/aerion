<script lang="ts">
  import { format, isSameDay, differenceInMinutes, startOfDay, addMinutes, setHours, setMinutes } from 'date-fns'
  import Icon from '@iconify/svelte'
  import { onMount, onDestroy } from 'svelte'
  import EventDialog from './EventDialog.svelte'

  interface Props {
    date: Date
    events: any[]
    calendars: any[]
    onClose: () => void
    onOpenEvent: (event: any) => void
    onRefresh: () => void
  }

  let { date, events, calendars, onClose, onOpenEvent, onRefresh }: Props = $props()

  let currentTime = $state(new Date())
  let timer: any
  let showEventDialog = $state(false)
  let initialEventDate = $state<Date | undefined>(undefined)

  onMount(() => {
    timer = setInterval(() => {
      currentTime = new Date()
    }, 60000)
  })

  onDestroy(() => {
    if (timer) clearInterval(timer)
  })

  const dayEvents = $derived(() => {
    return events
      .filter(e => isSameDay(new Date(e.startTime), date))
      .sort((a, b) => new Date(a.startTime).getTime() - new Date(b.startTime).getTime())
  })

  const isToday = $derived(isSameDay(date, new Date()))

  const hours = Array.from({ length: 24 }, (_, i) => i)

  function handleGridClick(e: MouseEvent) {
    const rect = (e.currentTarget as HTMLElement).getBoundingClientRect()
    const y = e.clientY - rect.top
    const totalMinutes = (y / 60) * 60
    const hour = Math.floor(totalMinutes / 60)
    const minute = Math.floor(totalMinutes % 60)
    
    initialEventDate = setMinutes(setHours(startOfDay(date), hour), minute)
    showEventDialog = true
  }

  function getEventStyle(event: any) {
    const start = new Date(event.startTime)
    const end = new Date(event.endTime)
    const startMins = differenceInMinutes(start, startOfDay(start))
    const duration = differenceInMinutes(end, start)
    
    // Each hour is 60px height
    const top = (startMins / 60) * 60
    const height = Math.max((duration / 60) * 60, 20) // Minimum 20px height
    
    const cal = calendars.find(c => c.id === event.calendarId)
    const color = cal?.color || '#3b82f6'

    return `top: ${top}px; height: ${height}px; border-left-color: ${color}; background-color: ${color}22;`
  }

  const markerPosition = $derived(() => {
    if (!isToday) return -1
    const mins = differenceInMinutes(currentTime, startOfDay(currentTime))
    return (mins / 60) * 60
  })

  let scrollContainer: HTMLDivElement

  onMount(() => {
    if (isToday) {
      const top = markerPosition() - 200
      if (scrollContainer) {
        scrollContainer.scrollTop = Math.max(0, top)
      }
    } else if (dayEvents().length > 0) {
        const firstEventStart = new Date(dayEvents()[0].startTime)
        const mins = differenceInMinutes(firstEventStart, startOfDay(firstEventStart))
        const top = (mins / 60) * 60 - 100
        if (scrollContainer) {
            scrollContainer.scrollTop = Math.max(0, top)
        }
    }
  })
</script>

<div class="flex flex-col h-full bg-background animate-in fade-in slide-in-from-right-4 duration-300">
  <div class="flex items-center justify-between px-6 py-4 border-b border-border bg-muted/10">
    <div class="flex items-center gap-4">
      <button 
        onclick={onClose}
        class="p-2 hover:bg-muted rounded-full transition-colors"
        title="Back to Month View"
      >
        <Icon icon="mdi:arrow-left" class="w-5 h-5" />
      </button>
      <div>
        <h2 class="text-xl font-semibold">{format(date, 'EEEE, MMMM d, yyyy')}</h2>
        {#if isToday}
          <span class="text-xs font-medium text-primary uppercase tracking-wider">Today</span>
        {/if}
      </div>
    </div>
  </div>

  <div class="flex-1 overflow-y-auto relative" bind:this={scrollContainer}>
    <div class="flex min-h-full">
      <!-- Time Labels -->
      <div class="w-16 flex-shrink-0 border-r border-border bg-muted/5">
        {#each hours as hour}
          <div class="h-[60px] pr-2 text-right text-[10px] text-muted-foreground font-medium flex items-start justify-end pt-2">
            {format(addMinutes(startOfDay(date), hour * 60), 'HH:mm')}
          </div>
        {/each}
      </div>

      <!-- Grid and Events -->
      <div class="flex-1 relative">
        <!-- Horizontal grid lines -->
        {#each hours as hour}
          <div class="absolute left-0 right-0 border-b border-border/50 h-[60px]" style="top: {hour * 60}px"></div>
        {/each}

        <!-- All Day Events Header (Simplified for now) -->
        {#if dayEvents().some(e => e.isAllDay)}
            <div class="sticky top-0 z-20 bg-background/95 backdrop-blur border-b border-border p-2 space-y-1">
                {#each dayEvents().filter(e => e.isAllDay) as event}
                    <button 
                        class="w-full text-left px-3 py-1 text-xs rounded border bg-primary/10 border-primary/30 truncate"
                        onclick={() => onOpenEvent(event)}
                    >
                        <span class="font-semibold text-primary">All day:</span> {event.summary}
                    </button>
                {/each}
            </div>
        {/if}

        <!-- Timed Events -->
        <div 
          class="relative h-[1440px] cursor-pointer" 
          onclick={handleGridClick}
          role="button"
          tabindex="0"
          onkeydown={(e) => e.key === 'Enter' && handleGridClick(e as any)}
        >
          {#each dayEvents().filter(e => !e.isAllDay) as event}
            <button 
              class="absolute left-1 right-4 px-3 py-2 text-xs rounded border-l-4 shadow-sm hover:shadow-md hover:brightness-95 transition-all text-left flex flex-col gap-0.5 overflow-hidden z-10"
              style={getEventStyle(event)}
              onclick={(e) => { e.stopPropagation(); onOpenEvent(event) }}
            >
              <span class="font-bold truncate">{event.summary || 'No Subject'}</span>
              <span class="text-[10px] opacity-80">
                {format(new Date(event.startTime), 'HH:mm')} - {format(new Date(event.endTime), 'HH:mm')}
              </span>
              {#if event.location}
                <div class="flex items-center gap-1 text-[10px] opacity-80 truncate">
                  <Icon icon="mdi:map-marker" class="w-3 h-3" />
                  {event.location}
                </div>
              {/if}
            </button>
          {/each}

          <!-- Current Time Marker -->
          {#if isToday && markerPosition() >= 0}
            <div 
              class="absolute left-0 right-0 flex items-center z-30 pointer-events-none"
              style="top: {markerPosition()}px"
            >
              <div class="w-2 h-2 rounded-full bg-red-500 -ml-1"></div>
              <div class="flex-1 h-0.5 bg-red-500"></div>
            </div>
          {/if}
        </div>
      </div>
    </div>
  </div>
</div>

<EventDialog
  bind:open={showEventDialog}
  initialDate={initialEventDate}
  calendars={calendars}
  onSave={onRefresh}
/>
