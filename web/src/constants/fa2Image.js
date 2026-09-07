/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

export const FA2_IMAGE_MODEL_SPECS = Object.freeze({
  'gpt-image-2': Object.freeze({
    resolutions: Object.freeze(['1K', '2K', '4K']),
    defaultResolution: '2K',
    aspectRatios: Object.freeze([
      '3:1',
      '21:9',
      '2:1',
      '16:9',
      '3:2',
      '4:3',
      '5:4',
      '1:1',
      '4:5',
      '3:4',
      '2:3',
      '9:16',
      '1:2',
      '1:3',
    ]),
    maxImages: 17,
  }),
  'nano-banana-pro': Object.freeze({
    resolutions: Object.freeze(['1K', '2K', '4K']),
    defaultResolution: '1K',
    aspectRatios: Object.freeze([
      '21:9',
      '16:9',
      '3:2',
      '4:3',
      '5:4',
      '1:1',
      '4:5',
      '3:4',
      '2:3',
      '9:16',
    ]),
    maxImages: 10,
  }),
  'nano-banana2': Object.freeze({
    resolutions: Object.freeze(['1K', '2K', '4K']),
    defaultResolution: '1K',
    aspectRatios: Object.freeze([
      '21:9',
      '16:9',
      '3:2',
      '4:3',
      '5:4',
      '1:1',
      '4:5',
      '3:4',
      '2:3',
      '9:16',
    ]),
    maxImages: 14,
  }),
  'seedream-5-0': Object.freeze({
    resolutions: Object.freeze(['2K', '3K']),
    defaultResolution: '2K',
    aspectRatios: Object.freeze(['16:9', '4:3', '1:1', '3:4', '9:16']),
    maxImages: 14,
  }),
});

export const FA2_IMAGE_MODELS = Object.freeze(
  Object.keys(FA2_IMAGE_MODEL_SPECS),
);

export const isFa2ImageModel = (modelName) =>
  Object.prototype.hasOwnProperty.call(
    FA2_IMAGE_MODEL_SPECS,
    String(modelName || '').trim(),
  );

export const getFa2ImageModelSpec = (modelName) =>
  FA2_IMAGE_MODEL_SPECS[String(modelName || '').trim()] || null;

export const getFa2ImageAspectRatioOptions = (modelName) =>
  (getFa2ImageModelSpec(modelName)?.aspectRatios || []).map((value) => ({
    label: value,
    value,
  }));

export const getFa2ImageResolutionOptions = (modelName) =>
  (getFa2ImageModelSpec(modelName)?.resolutions || []).map((value) => ({
    label: value,
    value,
  }));
