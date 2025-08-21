import React, { createContext, useContext, useState, useEffect, useCallback, useRef } from 'react';
import { roadmapAPI, handleAPIError } from '../services/api';
import toast from 'react-hot-toast';

const RoadmapContext = createContext();

export const useRoadmap = () => {
  const context = useContext(RoadmapContext);
  if (!context) {
    throw new Error('useRoadmap must be used within a RoadmapProvider');
  }
  return context;
};

export const RoadmapProvider = ({ children }) => {
  const [roadmapData, setRoadmapData] = useState(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState(null);

  // Request deduplication to prevent redundant API calls
  const activeRequests = useRef(new Map());

  const deduplicateRequest = useCallback(async (key, requestFn) => {
    // If request is already in progress, return the existing promise
    if (activeRequests.current.has(key)) {
      console.log(`Deduplicating request: ${key}`);
      return activeRequests.current.get(key);
    }

    // Create new request
    const requestPromise = requestFn().finally(() => {
      // Clean up after request completes
      activeRequests.current.delete(key);
    });

    activeRequests.current.set(key, requestPromise);
    return requestPromise;
  }, []);

  // Load basic data first (fast) - simplified approach
  const loadBasicData = useCallback(async () => {
    try {
      const basicData = await roadmapAPI.getBasicData();
      console.log('Basic data loaded:', basicData); // Debug log

      // Set basic data immediately for fast UI rendering
      setRoadmapData(prevData => ({
        ...prevData,
        pillars: basicData.pillars.map(pillar => ({
          ...pillar,
          milestones: [] // Will be loaded with full data
        })),
        quarters: basicData.quarters,
        components: basicData.project?.components,
        versions: basicData.project?.versions,
        project: basicData.project,
      }));

      return basicData;
    } catch (error) {
      console.warn('Failed to load basic data, will load full data:', error);
      return null;
    }
  }, []);



  const loadEpics = useCallback(async (milestoneIds) => {
    const requestKey = `epics-${milestoneIds.sort().join(',')}`;
    return deduplicateRequest(requestKey, async () => {
      try {
        console.log('Fetching epic data for milestones:', milestoneIds); // Debug log
        const epics = await roadmapAPI.getEpics({ milestoneIds });
        console.log('Epics loaded:', epics); // Debug log

        setRoadmapData(prevData => {
          if (!prevData) return prevData;

          // Create a map of epics grouped by milestone_id for efficient lookup
          const epicsByMilestone = {};
          epics.epics.forEach(epic => {
            const milestoneIds = epic.milestone_ids; // Fixed: should be milestone_id, not milestone_ids
            if (milestoneIds) {
              milestoneIds.forEach(milestoneId => {
                if (!epicsByMilestone[milestoneId]) {
                  epicsByMilestone[milestoneId] = [];
                }
                epicsByMilestone[milestoneId].push(epic);
              });
              // if (!epicsByMilestone[milestoneId]) {
              //   epicsByMilestone[milestoneId] = [];
              // }
              // epicsByMilestone[milestoneId].push(epic);
            }
          });

          // Merge the new epics into the existing data
          const updatedPillars = prevData.pillars.map(pillar => ({
            ...pillar,
            milestones: pillar.milestones.map(milestone => {
              const milestoneEpics = epicsByMilestone[milestone.id] || [];

              // Merge existing epics with new ones, avoiding duplicates
              const existingEpicIds = new Set(milestone.epics.map(e => e.id));
              const newEpics = milestoneEpics.filter(e => !existingEpicIds.has(e.id));

              return {
                ...milestone,
                epics: [...milestone.epics, ...newEpics],
              };
            }),
          }));

          console.log("Updated pillars are", updatedPillars);

          return {
            ...prevData,
            pillars: updatedPillars,
          };
        });

        return epics;
      } catch (error) {
        const errorInfo = handleAPIError(error);
        console.error('Failed to load epics:', errorInfo);
        return null;
      }
    });
  }, [deduplicateRequest]);

  const loadMilestones = useCallback(async (pillarIds, quarters) => {
    const requestKey = `milestones-${pillarIds.sort().join(',')}-${quarters.sort().join(',')}`;
    return deduplicateRequest(requestKey, async () => {
      try {
        console.log('Fetching milestone data for data:', pillarIds, quarters); // Debug log
        const milestones = await roadmapAPI.getMilestones({ pillarIds, quarters });
        console.log('Milestones loaded:', milestones); // Debug log

        // Update the roadmap data with the loaded milestones
        setRoadmapData(prevData => {
          if (!prevData) return prevData;

          // Create a map of milestones grouped by pillar_id for efficient lookup
          const milestonesByPillar = {};
          milestones.milestones.forEach(milestone => {
            const pillarId = milestone.pillar_id;
            if (!milestonesByPillar[pillarId]) {
              milestonesByPillar[pillarId] = [];
            }
            milestonesByPillar[pillarId].push(milestone);
          });

          // Merge the new milestones into the existing data
          const updatedPillars = prevData.pillars.map(pillar => {
            const pillarMilestones = milestonesByPillar[pillar.id] || [];

            // Merge existing milestones with new ones, avoiding duplicates
            const existingMilestoneIds = new Set(pillar.milestones.map(m => m.id));
            const newMilestones = pillarMilestones.filter(m => !existingMilestoneIds.has(m.id));

            return {
              ...pillar,
              milestones: [...pillar.milestones, ...newMilestones],
            };
          });

          return {
            ...prevData,
            pillars: updatedPillars,
          };
        });
        return milestones;
      } catch (error) {
        const errorInfo = handleAPIError(error);
        console.error('Failed to load milestones:', errorInfo);
        return null;
      }
    });
  }, [deduplicateRequest]);

  // Load roadmap data with progressive loading
  const loadRoadmap = useCallback(async () => {
    try {
      setIsLoading(true);
      setError(null);

      // Step 1: Load basic data first for fast UI feedback
      const basicData = await loadBasicData();
      if (!basicData) {
        throw new Error('Failed to load basic data');
      }
      setIsLoading(false);

      // Step 2: Load milestones for all pillars
      const pillarIds = basicData.pillars.map(pillar => pillar.id);
      const quarters = basicData.quarters;

      console.log('Loading milestones for pillars:', pillarIds, 'quarters:', quarters);
      const milestones = await loadMilestones(pillarIds, quarters);

      console.log("Got milestones?", milestones)

      const milestoneIds = milestones.milestones.map(milestone => milestone.id);
      const epics = await loadEpics(milestoneIds);

      console.log("Got epics?", epics)

    } catch (error) {
      const errorInfo = handleAPIError(error);
      console.error('Failed to load roadmap:', errorInfo);
      setError(errorInfo.message);
      toast.error(`Failed to load roadmap: ${errorInfo.message}`);
    } finally {
      setIsLoading(false);
    }
  }, [loadBasicData, loadMilestones, loadEpics]);


  // Load roadmap on mount
  useEffect(() => {
    loadRoadmap();
  }, [loadRoadmap]);

  const refreshMilestoneEpics = useCallback(async (milestoneIds) => {
    const requestKey = `refresh-milestone-epics-${milestoneIds.sort().join(',')}`;
    return deduplicateRequest(requestKey, async () => {
      try {
        console.log('Refreshing epics for milestones:', milestoneIds);
        const epics = await roadmapAPI.getEpics({ milestoneIds });

        setRoadmapData(prevData => {
          if (!prevData) return prevData;

          // Create a map of epics by milestone
          const epicsByMilestone = {};
          epics.epics?.forEach(epic => {
            if (epic.milestone_ids) {
              epic.milestone_ids.forEach(milestoneId => {
                if (!epicsByMilestone[milestoneId]) {
                  epicsByMilestone[milestoneId] = [];
                }
                epicsByMilestone[milestoneId].push(epic);
              });
            }
          });

          const updatedPillars = prevData.pillars.map(pillar => ({
            ...pillar,
            milestones: pillar.milestones.map(milestone => {
              if (milestoneIds.includes(milestone.id)) {
                return {
                  ...milestone,
                  epics: epicsByMilestone[milestone.id] || [],
                };
              }
              return milestone;
            }),
          }));

          return {
            ...prevData,
            pillars: updatedPillars,
          };
        });

        return { success: true };
      } catch (error) {
        console.error('Failed to refresh milestone epics:', error);
        return { success: false, error: error.message };
      }
    });
  }, [deduplicateRequest]);

  // Smart refresh functions for targeted updates
  const refreshPillarMilestones = useCallback(async (pillarId) => {
    const requestKey = `refresh-pillar-milestones-${pillarId}`;
    return deduplicateRequest(requestKey, async () => {
      try {
        console.log('Refreshing milestones for pillar:', pillarId);
        const milestones = await roadmapAPI.getMilestones({ pillarIds: [pillarId] });

        setRoadmapData(prevData => {
          if (!prevData) return prevData;

          const updatedPillars = prevData.pillars.map(pillar => {
            if (pillar.id === pillarId) {
              return {
                ...pillar,
                milestones: milestones.milestones || [],
              };
            }
            return pillar;
          });

          return {
            ...prevData,
            pillars: updatedPillars,
          };
        });

        // Load epics for the refreshed milestones
        const milestoneIds = milestones.milestones?.map(m => m.id) || [];
        if (milestoneIds.length > 0) {
          await refreshMilestoneEpics(milestoneIds);
        }

        return { success: true };
      } catch (error) {
        console.error('Failed to refresh pillar milestones:', error);
        return { success: false, error: error.message };
      }
    });
  }, [deduplicateRequest, refreshMilestoneEpics]);


  // Batch refresh function for multiple operations
  const batchRefresh = useCallback(async (operations) => {
    const pillarIds = new Set();
    const milestoneIds = new Set();

    operations.forEach(op => {
      if (op.type === 'pillar' && op.id) pillarIds.add(op.id);
      if (op.type === 'milestone' && op.id) milestoneIds.add(op.id);
    });

    // Refresh pillars first, then milestones
    const refreshPromises = [];

    if (pillarIds.size > 0) {
      pillarIds.forEach(pillarId => {
        refreshPromises.push(refreshPillarMilestones(pillarId));
      });
    }

    if (milestoneIds.size > 0) {
      refreshPromises.push(refreshMilestoneEpics(Array.from(milestoneIds)));
    }

    await Promise.all(refreshPromises);
  }, [refreshPillarMilestones, refreshMilestoneEpics]);

  // Create milestone with smart refresh
  const createMilestone = async (milestoneData) => {
    try {
      const milestone = await roadmapAPI.createMilestone(milestoneData);

      // Smart refresh: reload milestones for the affected pillar
      await refreshPillarMilestones(milestoneData.pillar_id);

      toast.success('Milestone created successfully!');
      return { success: true, data: milestone };
    } catch (error) {
      const errorInfo = handleAPIError(error);
      toast.error(`Failed to create milestone: ${errorInfo.message}`);
      return { success: false, error: errorInfo.message };
    }
  };

  // Update milestone with smart partial refresh
  const updateMilestone = async (milestoneId, milestoneData) => {
    try {
      const response = await roadmapAPI.updateMilestone(milestoneId, milestoneData);

      // Smart update: only refresh the affected pillar's milestones
      setRoadmapData(prevData => {
        if (!prevData) return prevData;

        const updatedPillars = prevData.pillars.map(pillar => ({
          ...pillar,
          milestones: pillar.milestones.map(milestone => {
            if (milestone.id === milestoneId) {
              return {
                ...milestone,
                name: milestoneData.name,
                quarter: milestoneData.quarter,
              };
            }
            return milestone;
          }),
        }));

        return {
          ...prevData,
          pillars: updatedPillars,
        };
      });

      toast.success('Milestone updated successfully!');
      return { success: true, data: response };
    } catch (error) {
      const errorInfo = handleAPIError(error);
      toast.error(`Failed to update milestone: ${errorInfo.message}`);
      return { success: false, error: errorInfo.message };
    }
  };

  // Create epic with smart refresh
  const createEpic = async (epicData) => {
    try {
      const epic = await roadmapAPI.createEpic(epicData);

      // Smart refresh: reload epics for the affected milestone
      await refreshMilestoneEpics([epicData.milestone_id]);

      toast.success('Epic created successfully!');
      return { success: true, data: epic };
    } catch (error) {
      const errorInfo = handleAPIError(error);
      toast.error(`Failed to create epic: ${errorInfo.message}`);
      return { success: false, error: errorInfo.message };
    }
  };

  // Update epic with smart partial refresh
  const updateEpic = async (epicId, epicData) => {
    try {
      const response = await roadmapAPI.updateEpic(epicId, epicData);

      // Smart update: only update the specific epic in state
      setRoadmapData(prevData => {
        if (!prevData) return prevData;

        const updatedPillars = prevData.pillars.map(pillar => ({
          ...pillar,
          milestones: pillar.milestones.map(milestone => ({
            ...milestone,
            epics: milestone.epics.map(epic => {
              if (epic.id === epicId) {
                return {
                  ...epic,
                  name: epicData.name,
                  component: epicData.component,
                  version: epicData.version,
                  priority: epicData.priority,
                  // Note: assignee updates might need special handling
                };
              }
              return epic;
            }),
          })),
        }));

        return {
          ...prevData,
          pillars: updatedPillars,
        };
      });

      toast.success('Epic updated successfully!');
      return { success: true, data: response };
    } catch (error) {
      const errorInfo = handleAPIError(error);
      toast.error(`Failed to update epic: ${errorInfo.message}`);
      return { success: false, error: errorInfo.message };
    }
  };

  // Move epic to different milestone with smart refresh
  const moveEpic = async (epicId, newMilestoneId) => {
    try {
      await roadmapAPI.updateEpicMilestone(epicId, newMilestoneId);

      // Find the affected milestones and refresh them
      let oldMilestoneId = null;

      // First, find which milestone the epic was in
      roadmapData?.pillars?.forEach(pillar => {
        pillar.milestones?.forEach(milestone => {
          milestone.epics?.forEach(epic => {
            if (epic.id === epicId) {
              oldMilestoneId = milestone.id;
            }
          });
        });
      });

      // Refresh both the old and new milestones to ensure data consistency
      const milestonesToRefresh = [newMilestoneId];
      if (oldMilestoneId && oldMilestoneId !== newMilestoneId) {
        milestonesToRefresh.push(oldMilestoneId);
      }

      await refreshMilestoneEpics(milestonesToRefresh);

      toast.success('Epic moved successfully!');
      return { success: true };
    } catch (error) {
      const errorInfo = handleAPIError(error);
      toast.error(`Failed to move epic: ${errorInfo.message}`);

      // On error, refresh the affected milestones to ensure consistency
      if (roadmapData?.pillars) {
        const allMilestoneIds = roadmapData.pillars
          .flatMap(p => p.milestones || [])
          .map(m => m.id);
        await refreshMilestoneEpics(allMilestoneIds);
      }

      return { success: false, error: errorInfo.message };
    }
  };

  // Get component versions
  const getComponentVersions = async (componentName) => {
    try {
      const response = await roadmapAPI.getComponentVersions(componentName);
      // const
      return { success: true, data: response.versions };
    } catch (error) {
      const errorInfo = handleAPIError(error);
      return { success: false, error: errorInfo.message };
    }
  };

  // Get assignable users
  const getAssignableUsers = async (issueKey, query = '') => {
    try {
      const response = await roadmapAPI.getAssignableUsers(issueKey, query);
      return { success: true, data: response.users };
    } catch (error) {
      const errorInfo = handleAPIError(error);
      return { success: false, error: errorInfo.message };
    }
  };

  const value = {
    roadmapData,
    isLoading,
    error,
    loadRoadmap,
    loadBasicData,
    loadMilestones,
    loadEpics,
    refreshPillarMilestones,
    refreshMilestoneEpics,
    batchRefresh,
    createMilestone,
    updateMilestone,
    createEpic,
    updateEpic,
    moveEpic,
    getComponentVersions,
    getAssignableUsers,
  };

  return (
    <RoadmapContext.Provider value={value}>
      {children}
    </RoadmapContext.Provider>
  );
};
