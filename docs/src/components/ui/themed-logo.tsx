"use client";

export function ThemedLogo() {
  return (
    <div className="relative flex items-center justify-center size-8">
      <svg
        viewBox="0 0 32 32"
        fill="none"
        xmlns="http://www.w3.org/2000/svg"
        className="size-8"
        aria-hidden="true"
      >
        {/* Shield / lock symbol */}
        <rect
          x="2"
          y="2"
          width="28"
          height="28"
          rx="6"
          className="fill-indigo-500 dark:fill-indigo-400"
        />
        {/* Shield outline */}
        <path
          d="M16 7L9 10V15C9 19.42 11.87 23.53 16 25C20.13 23.53 23 19.42 23 15V10L16 7Z"
          className="fill-white/90"
        />
        {/* Lock body */}
        <rect x="13" y="15" width="6" height="5" rx="1" className="fill-indigo-500 dark:fill-indigo-400" />
        {/* Lock shackle */}
        <path
          d="M14 15V13C14 11.9 14.9 11 16 11C17.1 11 18 11.9 18 13V15"
          stroke="currentColor"
          strokeWidth="1.5"
          strokeLinecap="round"
          className="text-indigo-500 dark:text-indigo-400"
          fill="none"
        />
        {/* Keyhole */}
        <circle cx="16" cy="17.5" r="0.8" className="fill-white" />
      </svg>
    </div>
  );
}
