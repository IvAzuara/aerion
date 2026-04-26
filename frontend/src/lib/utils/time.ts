
/**
 * Parses a flexible time string into HH:mm format.
 * Supports formats like:
 * - 1343 -> 13:43
 * - 143pm -> 13:43
 * - 2p -> 14:00
 * - 9 -> 09:00
 * - 930 -> 09:30
 * - 1:45 PM -> 13:45
 */
export function parseFlexibleTime(input: string): string | null {
  if (!input) return null;
  
  let cleanInput = input.toLowerCase().trim();
  const isPM = cleanInput.includes('p');
  const isAM = cleanInput.includes('a');
  
  // Handle colon
  cleanInput = cleanInput.replace(':', '');
  
  // Remove all non-numeric characters for digits processing
  const digits = cleanInput.replace(/\D/g, '');
  
  if (!digits) return null;

  let hours = 0;
  let minutes = 0;

  if (digits.length === 1 || digits.length === 2) {
    // e.g., "9", "13", "2p"
    hours = parseInt(digits);
    minutes = 0;
  } else if (digits.length === 3) {
    // e.g., "143", "930"
    hours = parseInt(digits.substring(0, 1));
    minutes = parseInt(digits.substring(1));
  } else if (digits.length === 4) {
    // e.g., "1343", "0930"
    hours = parseInt(digits.substring(0, 2));
    minutes = parseInt(digits.substring(2));
  } else {
    return null;
  }

  if (isPM && hours < 12) hours += 12;
  if (isAM && hours === 12) hours = 0;
  
  // Cap values
  if (hours > 23) hours = 23;
  if (minutes > 59) minutes = 59;
  
  return `${hours.toString().padStart(2, '0')}:${minutes.toString().padStart(2, '0')}`;
}

/**
 * Generates time options in 15-minute intervals.
 */
export function generateTimeOptions(): string[] {
  return Array.from({ length: 96 }, (_, i) => {
    const hours = Math.floor(i / 4);
    const minutes = (i % 4) * 15;
    return `${hours.toString().padStart(2, '0')}:${minutes.toString().padStart(2, '0')}`;
  });
}

/**
 * Formats HH:mm into a more readable format if needed, 
 * but for this task we might want to keep it simple or follow existing patterns.
 */
export function formatTime(time: string): string {
  // For now just return as is, or maybe add AM/PM if we want to be fancy.
  // The current app uses 24h format.
  return time;
}
