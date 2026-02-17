/**
 * Class merging utility (from Velocity theme pattern).
 * Combines clsx for conditional classes with tailwind-merge
 * to resolve Tailwind class conflicts.
 */
import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";

export function cn(...inputs: ClassValue[]): string {
  return twMerge(clsx(inputs));
}
