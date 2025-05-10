'use client';

import { useState } from 'react';
import { InputSelect, type SelectOption } from '@/components/ui/input-select';
import { InputAddon } from '@/components/ui/input-addon';

export default function InputAddonsExample() {
  const [inputValue, setInputValue] = useState('');
  const [selectedOption, setSelectedOption] = useState('.degov.ai');

  const options: SelectOption[] = [
    { value: '.degov.ai', label: '.degov.ai' },
    { value: '.eth', label: '.eth' },
    { value: '.sol', label: '.sol' },
    { value: '.crypto', label: '.crypto' }
  ];

  return (
    <div className="container py-10">
      <div className="mx-auto max-w-2xl space-y-10">
        <h1 className="text-3xl font-bold">Input Addon Components</h1>

        <section className="space-y-6">
          <h2 className="text-2xl font-bold">InputAddon Component</h2>
          <p className="text-muted-foreground">
            A flexible input component that supports prefixes and suffixes
          </p>

          <div className="space-y-6">
            <div>
              <label className="mb-2 block font-medium">With Suffix</label>
              <InputAddon
                value={inputValue}
                onChange={(e) => setInputValue(e.target.value)}
                suffix=".degov.ai"
                placeholder="Enter name"
              />
            </div>

            <div>
              <label className="mb-2 block font-medium">With Prefix</label>
              <InputAddon
                value={inputValue}
                onChange={(e) => setInputValue(e.target.value)}
                prefix="https://"
                placeholder="Enter domain"
              />
            </div>

            <div>
              <label className="mb-2 block font-medium">With Both</label>
              <InputAddon
                value={inputValue}
                onChange={(e) => setInputValue(e.target.value)}
                prefix="https://"
                suffix=".com"
                placeholder="Enter domain"
              />
            </div>
          </div>
        </section>

        <section className="space-y-6">
          <h2 className="text-2xl font-bold">InputSelect Component</h2>
          <p className="text-muted-foreground">An input with a connected select dropdown</p>

          <div className="space-y-6">
            <div>
              <label className="mb-2 block font-medium">With Suffix Select</label>
              <InputSelect
                value={inputValue}
                onChange={(e) => setInputValue(e.target.value)}
                options={options}
                selectValue={selectedOption}
                onSelectChange={setSelectedOption}
                position="suffix"
                placeholder="Enter name"
              />
            </div>

            <div>
              <label className="mb-2 block font-medium">With Prefix Select</label>
              <InputSelect
                value={inputValue}
                onChange={(e) => setInputValue(e.target.value)}
                options={[
                  { value: 'https://', label: 'HTTPS' },
                  { value: 'http://', label: 'HTTP' },
                  { value: 'ftp://', label: 'FTP' }
                ]}
                selectValue="https://"
                onSelectChange={(v) => console.log(v)}
                position="prefix"
                placeholder="Enter domain"
              />
            </div>

            <div className="bg-muted rounded-md p-4">
              <h3 className="font-medium">Current Values:</h3>
              <p>Input: {inputValue}</p>
              <p>Selected Option: {selectedOption}</p>
              <p>
                Combined: {inputValue}
                {selectedOption}
              </p>
            </div>
          </div>
        </section>
      </div>
    </div>
  );
}
