import React, { createContext, useContext, useState, useEffect } from 'react';

const ThemeContext = createContext();

// Available themes
export const THEMES = {
  LIGHT: 'light',
  DARK: 'dark',
  OCEAN: 'ocean',
  FOREST: 'forest',
  SUNSET: 'sunset',
  MONOCHROME: 'monochrome',
  MELODEE: 'melodee',
};

// Theme configurations
export const themeConfig = {
  [THEMES.LIGHT]: {
    name: 'Light',
    mode: 'light',
    colors: {
      navbar: 'bg-blue-700',
      navbarText: 'text-white',
      navbarHover: 'hover:text-blue-200',
      background: 'bg-gray-100',
      surface: 'bg-white',
      text: 'text-gray-900',
      textSecondary: 'text-gray-600',
      border: 'border-gray-200',
      button: 'bg-blue-500 hover:bg-blue-600',
    },
  },
  [THEMES.DARK]: {
    name: 'Dark',
    mode: 'dark',
    colors: {
      navbar: 'bg-gray-800',
      navbarText: 'text-white',
      navbarHover: 'hover:text-blue-300',
      background: 'bg-gray-900',
      surface: 'bg-gray-800',
      text: 'text-gray-100',
      textSecondary: 'text-gray-400',
      border: 'border-gray-700',
      button: 'bg-blue-600 hover:bg-blue-700',
    },
  },
  [THEMES.OCEAN]: {
    name: 'Ocean',
    mode: 'light',
    colors: {
      navbar: 'bg-ocean-700',
      navbarText: 'text-white',
      navbarHover: 'hover:text-teal-200',
      background: 'bg-ocean-50',
      surface: 'bg-white',
      text: 'text-ocean-900',
      textSecondary: 'text-ocean-600',
      border: 'border-ocean-200',
      button: 'bg-teal-500 hover:bg-teal-600',
    },
  },
  [THEMES.FOREST]: {
    name: 'Forest',
    mode: 'light',
    colors: {
      navbar: 'bg-forest-700',
      navbarText: 'text-white',
      navbarHover: 'hover:text-forest-200',
      background: 'bg-forest-50',
      surface: 'bg-white',
      text: 'text-forest-900',
      textSecondary: 'text-brown-700',
      border: 'border-forest-200',
      button: 'bg-forest-600 hover:bg-forest-700',
    },
  },
  [THEMES.SUNSET]: {
    name: 'Sunset',
    mode: 'light',
    colors: {
      navbar: 'bg-sunset-600',
      navbarText: 'text-white',
      navbarHover: 'hover:text-sunset-100',
      background: 'bg-sunset-50',
      surface: 'bg-white',
      text: 'text-sunset-900',
      textSecondary: 'text-purple-700',
      border: 'border-sunset-200',
      button: 'bg-purple-600 hover:bg-purple-700',
    },
  },
  [THEMES.MONOCHROME]: {
    name: 'Monochrome',
    mode: 'light',
    colors: {
      navbar: 'bg-gray-800',
      navbarText: 'text-white',
      navbarHover: 'hover:text-gray-300',
      background: 'bg-gray-100',
      surface: 'bg-white',
      text: 'text-gray-900',
      textSecondary: 'text-gray-600',
      border: 'border-gray-300',
      button: 'bg-gray-700 hover:bg-gray-800',
    },
  },
  [THEMES.MELODEE]: {
    name: 'Melodee',
    mode: 'light',
    colors: {
      navbar: 'bg-gradient-to-r from-melodee-600 to-coral-500',
      navbarText: 'text-white',
      navbarHover: 'hover:text-melodee-100',
      background: 'bg-gradient-to-br from-purple-900 via-melodee-900 to-purple-950',
      surface: 'bg-white',
      text: 'text-melodee-900',
      textSecondary: 'text-coral-700',
      border: 'border-melodee-200',
      button: 'bg-melodee-600 hover:bg-melodee-700',
    },
  },
};

export function ThemeProvider({ children }) {
  // Initialize theme from localStorage or default to 'light'
  const [theme, setTheme] = useState(() => {
    const savedTheme = localStorage.getItem('melodee-theme');
    return savedTheme || THEMES.LIGHT;
  });

  // Apply theme class to document root whenever theme changes
  useEffect(() => {
    const root = window.document.documentElement;
    
    // Remove all theme classes first
    Object.values(THEMES).forEach(t => {
      root.classList.remove(t);
    });
    
    // Add dark class for dark mode (for existing dark: variants)
    const currentThemeConfig = themeConfig[theme];
    if (currentThemeConfig?.mode === 'dark') {
      root.classList.add('dark');
    } else {
      root.classList.remove('dark');
    }
    
    // Add the current theme class
    root.classList.add(theme);
    
    // Save to localStorage
    localStorage.setItem('melodee-theme', theme);
  }, [theme]);

  const toggleTheme = () => {
    setTheme((prevTheme) => (prevTheme === THEMES.LIGHT ? THEMES.DARK : THEMES.LIGHT));
  };

  const value = {
    theme,
    setTheme,
    toggleTheme,
    isDark: theme === THEMES.DARK,
    currentTheme: themeConfig[theme],
    availableThemes: THEMES,
  };

  return (
    <ThemeContext.Provider value={value}>
      {children}
    </ThemeContext.Provider>
  );
}

export function useTheme() {
  const context = useContext(ThemeContext);
  if (context === undefined) {
    throw new Error('useTheme must be used within a ThemeProvider');
  }
  return context;
}
