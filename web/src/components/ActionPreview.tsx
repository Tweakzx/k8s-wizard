import React, { useState } from 'react';
import type { ActionPreview as ActionPreviewType } from '../types';

interface ActionPreviewProps {
  preview: ActionPreviewType;
  onConfirm: () => void;
  onCancel?: () => void;
}

export const ActionPreview: React.FC<ActionPreviewProps> = ({ preview, onConfirm, onCancel }) => {
  const [showYaml, setShowYaml] = useState(false);

  const dangerColors = {
    low: 'bg-green-100 text-green-800 border-green-200',
    medium: 'bg-yellow-100 text-yellow-800 border-yellow-200',
    high: 'bg-red-100 text-red-800 border-red-200',
  };

  const dangerLabels = {
    low: '安全操作',
    medium: '谨慎操作',
    high: '危险操作',
  };

  return (
    <div className="bg-white rounded-xl border border-gray-200 shadow-lg overflow-hidden">
      {/* Header */}
      <div className="px-4 py-3 border-b border-gray-100 flex items-center justify-between">
        <div className="flex items-center gap-2">
          <span className="text-lg">
            {preview.type === 'create' && '📦'}
            {preview.type === 'scale' && '⚙️'}
            {preview.type === 'delete' && '🗑️'}
            {preview.type === 'get' && '🔍'}
          </span>
          <span className="font-semibold text-gray-800">操作预览</span>
        </div>
        <span className={`px-2 py-1 text-xs font-medium rounded-full border ${dangerColors[preview.dangerLevel]}`}>
          {dangerLabels[preview.dangerLevel]}
        </span>
      </div>

      {/* Content */}
      <div className="p-4">
        {/* Summary */}
        <div className="mb-4">
          <p className="text-gray-700">{preview.summary}</p>
        </div>

        {/* Resource Info */}
        <div className="flex items-center gap-4 text-sm text-gray-500 mb-4">
          <span>📍 {preview.resource}</span>
          <span>📁 {preview.namespace}</span>
        </div>

        {/* YAML Preview */}
        {preview.yaml && (
          <div className="mb-4">
            <button
              onClick={() => setShowYaml(!showYaml)}
              className="flex items-center gap-1 text-sm text-primary-600 hover:text-primary-700"
            >
              <svg
                className={`w-4 h-4 transition-transform ${showYaml ? 'rotate-90' : ''}`}
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
              </svg>
              查看 YAML
            </button>

            {showYaml && (
              <pre className="mt-2 p-3 bg-gray-900 text-gray-100 rounded-lg text-xs overflow-x-auto">
                <code>{preview.yaml}</code>
              </pre>
            )}
          </div>
        )}

        {/* Actions */}
        <div className="flex justify-end gap-3 pt-4 border-t border-gray-100">
          {onCancel && (
            <button
              onClick={onCancel}
              className="px-4 py-2 text-gray-600 hover:text-gray-800 transition-colors"
            >
              取消
            </button>
          )}
          <button
            onClick={onConfirm}
            className={`px-6 py-2 rounded-lg font-medium transition-colors ${
              preview.dangerLevel === 'high'
                ? 'bg-red-500 text-white hover:bg-red-600'
                : 'bg-primary-500 text-white hover:bg-primary-600'
            }`}
          >
            {preview.dangerLevel === 'high' ? '确认删除' : '确认执行'}
          </button>
        </div>
      </div>
    </div>
  );
};
