import React, { useState, useRef, useEffect } from 'react';
import { useTheme, THEMES, themeConfig } from '../context/ThemeContext';

function ThemeSelector() {
  const { theme, setTheme } = useTheme();
  const [isOpen, setIsOpen] = useState(false);
  const dropdownRef = useRef(null);

  // Close dropdown when clicking outside
  useEffect(() => {
    function handleClickOutside(event) {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target)) {
        setIsOpen(false);
      }
    }

    if (isOpen) {
      document.addEventListener('mousedown', handleClickOutside);
      return () => document.removeEventListener('mousedown', handleClickOutside);
    }
  }, [isOpen]);

  const handleThemeChange = (newTheme) => {
    setTheme(newTheme);
    setIsOpen(false);
  };

  return (
    <div className="relative" ref={dropdownRef}>
      <button
        onClick={() => setIsOpen(!isOpen)}
        className="p-2 rounded-lg bg-white/10 hover:bg-white/20 text-white transition-colors"
        title="Select Theme"
      >
        <svg
          className="w-5 h-5"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M7 21a4 4 0 01-4-4V5a2 2 0 012-2h4a2 2 0 012 2v12a4 4 0 01-4 4zm0 0h12a2 2 0 002-2v-4a2 2 0 00-2-2h-2.343M11 7.343l1.657-1.657a2 2 0 012.828 0l2.829 2.829a2 2 0 010 2.828l-8.486 8.485M7 17h.01"
          />
        </svg>
      </button>

      {isOpen && (
        <div 
          className="absolute right-0 mt-2 w-48 rounded-lg shadow-xl py-1 z-50"
          style={{ backgroundColor: '#ffffff', border: '1px solid #e5e7eb' }}
        >
          <div 
            className="px-3 py-2 text-xs font-semibold uppercase tracking-wider border-b"
            style={{ color: '#6b7280', borderColor: '#e5e7eb' }}
          >
            Select Theme
          </div>
          {Object.entries(THEMES).map(([key, value]) => (
            <button
              key={value}
              onClick={() => handleThemeChange(value)}
              className="w-full text-left px-4 py-2 text-sm transition-colors flex items-center justify-between"
              style={{
                backgroundColor: theme === value ? '#dbeafe' : 'transparent',
                color: theme === value ? '#1d4ed8' : '#374151',
                fontWeight: theme === value ? '500' : '400',
              }}
              onMouseEnter={(e) => {
                if (theme !== value) {
                  e.currentTarget.style.backgroundColor = '#f3f4f6';
                }
              }}
              onMouseLeave={(e) => {
                if (theme !== value) {
                  e.currentTarget.style.backgroundColor = 'transparent';
                }
              }}
            >
              <span>{themeConfig[value].name}</span>
              {theme === value && (
                <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
                  <path
                    fillRule="evenodd"
                    d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z"
                    clipRule="evenodd"
                  />
                </svg>
              )}
            </button>
          ))}
        </div>
      )}
    </div>
  );
}

export default ThemeSelector;
