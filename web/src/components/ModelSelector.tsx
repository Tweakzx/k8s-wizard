import React, { useState, useEffect, useRef } from 'react';
import { api } from '../services/api';
import type { ModelInfo } from '../types';

interface ModelSelectorProps {
  onModelChange?: (model: string) => void;
}

export const ModelSelector: React.FC<ModelSelectorProps> = ({ onModelChange }) => {
  const [currentModel, setCurrentModel] = useState<string>('');
  const [models, setModels] = useState<ModelInfo[]>([]);
  const [isOpen, setIsOpen] = useState(false);
  const [loading, setLoading] = useState(false);
  const dropdownRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    fetchModelInfo();
  }, []);

  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        setIsOpen(false);
      }
    };

    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  const fetchModelInfo = async () => {
    try {
      const info = await api.getModelInfo();
      setCurrentModel(info.current);
      setModels(info.models);
    } catch (error) {
      console.error('Failed to fetch model info:', error);
    }
  };

  const handleModelSelect = async (modelString: string) => {
    if (modelString === currentModel) {
      setIsOpen(false);
      return;
    }

    setLoading(true);
    try {
      const result = await api.setModel(modelString);
      if (result.success) {
        setCurrentModel(result.model);
        onModelChange?.(result.model);
      }
    } catch (error) {
      console.error('Failed to switch model:', error);
      alert(error instanceof Error ? error.message : '切换模型失败');
    } finally {
      setLoading(false);
      setIsOpen(false);
    }
  };

  // 按提供商分组
  const groupedModels = models.reduce((acc, model) => {
    if (!acc[model.provider]) {
      acc[model.provider] = [];
    }
    acc[model.provider].push(model);
    return acc;
  }, {} as Record<string, ModelInfo[]>);

  const providerNames: Record<string, string> = {
    glm: '智谱 GLM',
    deepseek: 'DeepSeek',
    claude: 'Claude',
  };

  return (
    <div className="relative" ref={dropdownRef}>
      <button
        onClick={() => !loading && setIsOpen(!isOpen)}
        disabled={loading}
        className={`flex items-center gap-2 px-3 py-1.5 bg-gray-100 rounded-full transition-all duration-200 ${
          loading ? 'opacity-50 cursor-wait' : 'hover:bg-gray-200 cursor-pointer'
        }`}
        title="点击切换模型"
      >
        <span className="text-sm font-medium text-gray-600">
          🤖 {loading ? '切换中...' : currentModel}
        </span>
        <svg
          className={`w-4 h-4 text-gray-400 transition-transform duration-200 ${isOpen ? 'rotate-180' : ''}`}
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
        </svg>
      </button>

      {isOpen && (
        <div className="absolute right-0 mt-2 w-64 bg-white rounded-lg shadow-lg border border-gray-200 z-50 max-h-80 overflow-y-auto">
          <div className="p-2">
            <div className="text-xs text-gray-500 px-2 py-1 border-b border-gray-100 mb-1">
              选择模型
            </div>
            {Object.entries(groupedModels).map(([provider, providerModels]) => (
              <div key={provider} className="mb-2">
                <div className="text-xs font-medium text-gray-400 px-2 py-1 uppercase">
                  {providerNames[provider] || provider}
                </div>
                {providerModels.map((model) => {
                  const modelString = `${model.provider}/${model.id}`;
                  const isSelected = modelString === currentModel;
                  return (
                    <button
                      key={modelString}
                      onClick={() => handleModelSelect(modelString)}
                      disabled={loading}
                      className={`w-full text-left px-3 py-2 rounded-md text-sm transition-colors ${
                        isSelected
                          ? 'bg-primary-100 text-primary-700 font-medium'
                          : 'hover:bg-gray-100 text-gray-700'
                      }`}
                    >
                      <div className="flex items-center justify-between">
                        <span>{model.name}</span>
                        {isSelected && (
                          <svg className="w-4 h-4 text-primary-600" fill="currentColor" viewBox="0 0 20 20">
                            <path
                              fillRule="evenodd"
                              d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z"
                              clipRule="evenodd"
                            />
                          </svg>
                        )}
                      </div>
                      <div className="text-xs text-gray-400 mt-0.5">{model.id}</div>
                    </button>
                  );
                })}
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
};
