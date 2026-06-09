import { ref } from 'vue';
import api from '../api';

const regions = ref<string[]>([]);
const loading = ref(false);
const loaded = ref(false);

export function useRegions() {
  const fetchRegions = async (force = false) => {
    if ((loaded.value && !force) || loading.value) return;
    loading.value = true;
    try {
      const { data } = await api.get('/regions');
      regions.value = Array.isArray(data?.items) ? data.items : [];
      loaded.value = true;
    } catch (error) {
      console.error('Failed to load regions', error);
      regions.value = [];
      loaded.value = false;
    } finally {
      loading.value = false;
    }
  };

  const resetRegionsCache = () => {
    loaded.value = false;
  };

  return {
    regions,
    loading,
    fetchRegions,
    resetRegionsCache,
  };
}
