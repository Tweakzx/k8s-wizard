import React, { useState } from 'react';
import type { ClarificationRequest, ClarificationField } from '../types';

interface ActionFormProps {
  clarification: ClarificationRequest;
  onSubmit: (formData: Record<string, any>) => void;
  onCancel?: () => void;
}

export const ActionForm: React.FC<ActionFormProps> = ({ clarification, onSubmit, onCancel }) => {
  const [formData, setFormData] = useState<Record<string, any>>(() => {
    const initial: Record<string, any> = {};
    clarification.fields.forEach((field) => {
      if (field.default !== undefined) {
        initial[field.key] = field.default;
      }
    });
    return initial;
  });

  const handleChange = (key: string, value: any) => {
    setFormData((prev) => ({ ...prev, [key]: value }));
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    onSubmit(formData);
  };

  const renderField = (field: ClarificationField) => {
    const value = formData[field.key] ?? '';

    switch (field.type) {
      case 'select':
        return (
          <select
            value={value}
            onChange={(e) => handleChange(field.key, e.target.value)}
            className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-transparent"
          >
            {field.options?.map((opt) => (
              <option key={opt.value} value={opt.value}>
                {opt.label}
              </option>
            ))}
          </select>
        );

      case 'number':
        const numValue = typeof value === 'number' ? value : parseInt(value) || 1;
        const minVal = 1;
        // 无上限，由用户根据集群资源自行决定
        
        const handleDecrement = () => {
          const newVal = Math.max(minVal, numValue - 1);
          handleChange(field.key, newVal);
        };
        
        const handleIncrement = () => {
          handleChange(field.key, numValue + 1);
        };
        
        return (
          <div className="flex items-center gap-3">
            {/* 减号按钮 */}
            <button
              type="button"
              onClick={handleDecrement}
              disabled={numValue <= minVal}
              className="w-10 h-10 flex items-center justify-center rounded-lg border border-gray-300 bg-white text-gray-600 hover:bg-gray-50 hover:border-gray-400 disabled:opacity-40 disabled:cursor-not-allowed transition-all"
            >
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M20 12H4" />
              </svg>
            </button>
            
            {/* 数字输入框 */}
            <div className="relative">
              <input
                type="number"
                value={numValue}
                onChange={(e) => {
                  const val = parseInt(e.target.value) || minVal;
                  handleChange(field.key, Math.max(minVal, val));
                }}
                min={minVal}
                className="w-20 px-3 py-2 text-center border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-transparent [appearance:textfield] [&::-webkit-outer-spin-button]:appearance-none [&::-webkit-inner-spin-button]:appearance-none"
              />
            </div>
            
            {/* 加号按钮 */}
            <button
              type="button"
              onClick={handleIncrement}
              className="w-10 h-10 flex items-center justify-center rounded-lg border border-gray-300 bg-white text-gray-600 hover:bg-gray-50 hover:border-gray-400 transition-all"
            >
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
              </svg>
            </button>
            
            {/* 快捷按钮 */}
            <div className="flex gap-1 ml-2">
              {[3, 5, 10].map((n) => (
                <button
                  key={n}
                  type="button"
                  onClick={() => handleChange(field.key, n)}
                  className={`px-3 py-2 text-sm rounded-md transition-colors ${
                    numValue === n
                      ? 'bg-primary-500 text-white border-primary-500'
                      : 'bg-gray-50 text-gray-600 border border-gray-200 hover:bg-gray-100'
                  }`}
                >
                  {n}
                </button>
              ))}
            </div>
          </div>
        );

      default:
        return (
          <input
            type="text"
            value={value}
            onChange={(e) => handleChange(field.key, e.target.value)}
            placeholder={field.placeholder}
            className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-transparent"
          />
        );
    }
  };

  // 按组分组字段
  const basicFields = clarification.fields.filter((f) => f.group !== 'advanced');
  const advancedFields = clarification.fields.filter((f) => f.group === 'advanced');
  const [showAdvanced, setShowAdvanced] = useState(false);

  return (
    <div className="bg-white rounded-xl border border-gray-200 shadow-lg overflow-hidden">
      {/* Header */}
      <div className="bg-gradient-to-r from-primary-500 to-primary-600 px-4 py-3">
        <h3 className="text-white font-semibold text-lg">{clarification.title}</h3>
        {clarification.description && (
          <p className="text-primary-100 text-sm mt-1">{clarification.description}</p>
        )}
      </div>

      {/* Form */}
      <form onSubmit={handleSubmit} className="p-4 space-y-4">
        {/* Basic Fields */}
        {basicFields.map((field) => (
          <div key={field.key}>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              {field.label}
              {field.required && <span className="text-red-500 ml-1">*</span>}
            </label>
            {renderField(field)}
          </div>
        ))}

        {/* Advanced Fields Toggle */}
        {advancedFields.length > 0 && (
          <div>
            <button
              type="button"
              onClick={() => setShowAdvanced(!showAdvanced)}
              className="flex items-center gap-1 text-sm text-gray-500 hover:text-gray-700"
            >
              <svg
                className={`w-4 h-4 transition-transform ${showAdvanced ? 'rotate-90' : ''}`}
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
              </svg>
              高级选项
            </button>

            {showAdvanced && (
              <div className="mt-3 space-y-4 pl-4 border-l-2 border-gray-200">
                {advancedFields.map((field) => (
                  <div key={field.key}>
                    <label className="block text-sm font-medium text-gray-700 mb-1">
                      {field.label}
                      {field.required && <span className="text-red-500 ml-1">*</span>}
                    </label>
                    {renderField(field)}
                  </div>
                ))}
              </div>
            )}
          </div>
        )}

        {/* Actions */}
        <div className="flex justify-end gap-3 pt-4 border-t border-gray-100">
          {onCancel && (
            <button
              type="button"
              onClick={onCancel}
              className="px-4 py-2 text-gray-600 hover:text-gray-800 transition-colors"
            >
              取消
            </button>
          )}
          <button
            type="submit"
            className="px-6 py-2 bg-primary-500 text-white rounded-lg hover:bg-primary-600 transition-colors font-medium"
          >
            确认
          </button>
        </div>
      </form>
    </div>
  );
};
