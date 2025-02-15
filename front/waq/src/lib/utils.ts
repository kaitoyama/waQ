import { type ClassValue, clsx } from "clsx"
import { twMerge } from "tailwind-merge"

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

export const backendURL = process.env.NEXT_PUBLIC_BACKEND_URL || "http://localhost:8080";
export const secretKey = process.env.NEXT_PUBLIC_SECRET

