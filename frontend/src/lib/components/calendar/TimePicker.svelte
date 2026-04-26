<script lang="ts">
  import { parseFlexibleTime, generateTimeOptions } from '$lib/utils/time'
  import { Input } from '$lib/components/ui/input'
  import Icon from '@iconify/svelte'
  import { tick } from 'svelte'

  interface Props {
    value: string
    disabled?: boolean
    class?: string
    options?: string[]
  }

  let { 
    value = $bindable(), 
    disabled = false, 
    class: className = '',
    options = generateTimeOptions()
  }: Props = $props()
  
  let inputValue = $state(value)
  let open = $state(false)
  let isFocused = $state(false)
  let dropdownContent: HTMLDivElement | null = $state(null)

  // Only sync external value to input when not focused to avoid interrupting typing
  $effect(() => {
    if (!isFocused) {
      inputValue = value
    }
  })

  function handleInput(e: Event) {
    const target = e.target as HTMLInputElement
    inputValue = target.value
    // We don't update 'value' here yet to allow the user to type freely
    // even if the intermediate state is not a valid time
  }

  function handleBlur() {
    isFocused = false
    // Small delay to allow selectOption to fire if it was a click on an option
    setTimeout(() => {
      const parsed = parseFlexibleTime(inputValue)
      if (parsed) {
        value = parsed
        inputValue = parsed
      } else {
        // If invalid, revert to last valid value
        inputValue = value
      }
      open = false
    }, 200)
  }

  function selectOption(opt: string) {
    value = opt
    inputValue = opt
    open = false
  }

  async function handleFocus() {
    if (disabled) return
    isFocused = true
    open = true
    await tick()
    scrollToSelected()
  }

  function scrollToSelected() {
    if (dropdownContent) {
      const selected = dropdownContent.querySelector('[data-selected="true"]')
      if (selected) {
        selected.scrollIntoView({ block: 'center' })
      }
    }
  }

  function handleKeyDown(e: KeyboardEvent) {
    if (e.key === 'Enter') {
      const parsed = parseFlexibleTime(inputValue)
      if (parsed) {
        value = parsed
        inputValue = parsed
      }
      open = false
      // @ts-ignore
      e.target.blur()
    } else if (e.key === 'Escape') {
      open = false
      // @ts-ignore
      e.target.blur()
    }
    
    // Stop propagation to prevent global shortcuts from intercepting
    e.stopPropagation()
  }
</script>

<div class="relative w-full {className}">
  <div class="relative w-full">
    <Icon icon="mdi:clock-outline" class="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 opacity-50 z-10" />
    <Input
      class="pl-9 {disabled ? 'opacity-50 grayscale cursor-not-allowed' : ''}"
      {disabled}
      bind:value={inputValue}
      oninput={handleInput}
      onblur={handleBlur}
      onfocus={handleFocus}
      onkeydown={handleKeyDown}
      onclick={() => !disabled && (open = true)}
      placeholder="00:00"
    />
  </div>

  {#if open && !disabled}
    <div 
      class="absolute z-50 mt-1 w-[120px] max-h-[300px] overflow-y-auto p-1 bg-popover text-popover-foreground rounded-md border shadow-md"
      bind:this={dropdownContent}
    >
      {#each options as opt}
        <button
          type="button"
          class="w-full text-left px-2 py-1.5 text-sm rounded-sm hover:bg-accent hover:text-accent-foreground transition-colors {value === opt ? 'bg-primary/10 text-primary' : ''}"
          onclick={() => selectOption(opt)}
          data-selected={value === opt}
        >
          {opt}
        </button>
      {/each}
    </div>
  {/if}
</div>

{#if open}
  <div 
    class="fixed inset-0 z-40" 
    onclick={() => open = false}
    aria-hidden="true"
  ></div>
{/if}
