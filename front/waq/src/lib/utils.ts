import { type ClassValue, clsx } from "clsx"
import { twMerge } from "tailwind-merge"

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

export const backendURL = process.env.NODE_ENV_BACKEND_URL || "http://localhost:8080";

