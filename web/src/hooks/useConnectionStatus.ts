import { useState, useEffect } from 'react';
import { api } from '../services/api';

export function useConnectionStatus(): boolean {
  const [connected, setConnected] = useState(false);

  useEffect(() => {
    const checkStatus = async () => {
      const isConnected = await api.checkHealth();
      setConnected(isConnected);
    };

    checkStatus();
    const interval = setInterval(checkStatus, 30000);
    return () => clearInterval(interval);
  }, []);

  return connected;
}
