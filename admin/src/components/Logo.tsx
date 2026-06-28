export default function Logo({ size = 32 }: { size?: number }) {
  return (
    <svg width={size} height={size} viewBox="0 0 32 32" fill="none" xmlns="http://www.w3.org/2000/svg">
      <rect width="32" height="32" rx="8" fill="url(#paint0_linear)" />
      <path d="M16 8L24 22H19.5L16 15L12.5 22H8L16 8Z" fill="white" />
      <path d="M16 26L19 21H13L16 26Z" fill="rgba(255, 255, 255, 0.6)" />
      <defs>
        <linearGradient id="paint0_linear" x1="0" y1="0" x2="32" y2="32" gradientUnits="userSpaceOnUse">
          <stop stopColor="#6366F1" />
          <stop offset="1" stopColor="#A855F7" />
        </linearGradient>
      </defs>
    </svg>
  )
}
