import React, { createContext, useContext, useState, useEffect, useCallback } from 'react';
import { roadmapAPI, handleAPIError } from '../services/api';
import { sortRoadmapData } from '../utils/sortingUtils';
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
        components: basicData.components,
        versions: basicData.versions,
      }));

      return basicData;
    } catch (error) {
      console.warn('Failed to load basic data, will load full data:', error);
      return null;
    }
  }, []);



  const loadEpics = useCallback(async (milestoneIds) => {
    try {
      console.log('Fetching epic data for milestones:', milestoneIds); // Debug log
      const epics = await roadmapAPI.getEpics({ milestoneIds });
      console.log('Epics loaded:', epics); // Debug log
      setRoadmapData(prevData => {
        if (!prevData) return prevData;

        // Create a map of epics grouped by milestone_id for efficient lookup
        const epicsByMilestone = {};
        epics.epics.forEach(epic => {
          const milestoneId = epic.milestone_ids;
          milestoneId.forEach(id => {
            if (!epicsByMilestone[id]) {
              epicsByMilestone[id] = [];
            }
            epicsByMilestone[id].push(epic);
          });
          // if (!epicsByMilestone[milestoneId]) {
          //   epicsByMilestone[milestoneId] = [];
          // }
          // epicsByMilestone[milestoneId].push(epic);
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
    }
    return null;
  }, []);

  const loadMilestones = useCallback(async (pillarIds, quarters) => {
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
    }
    return null;
  }, []);

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

      // Step 3: Get milestone IDs from current state to load epics
      // We need to access the updated state after milestones are loaded
      // setRoadmapData(prevData => {
      //   if (prevData && prevData.pillars) {
      //     // Collect all milestone IDs from the loaded milestones
      //     const milestoneIds = [];
      //     prevData.pillars.forEach(pillar => {
      //       pillar.milestones.forEach(milestone => {
      //         milestoneIds.push(milestone.id);
      //       });
      //     });

      //     if (milestoneIds.length > 0) {
      //       console.log('Loading epics for milestones:', milestoneIds);
      //       // Load epics asynchronously without blocking the UI
      //       loadEpics(milestoneIds).catch(error => {
      //         console.error('Failed to load epics:', error);
      //       });
      //     }
      //   }
      //   return prevData;
      // });

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

  // Create milestone
  const createMilestone = async (milestoneData) => {
    try {
      const milestone = await roadmapAPI.createMilestone(milestoneData);

      // Optimistically update the roadmap data
      setRoadmapData(prevData => {
        if (!prevData) return prevData;

        const updatedPillars = prevData.pillars.map(pillar => {
          if (pillar.id === milestoneData.pillar_id) {
            return {
              ...pillar,
              milestones: [...pillar.milestones, milestone],
            };
          }
          return pillar;
        });

        return {
          ...prevData,
          pillars: updatedPillars,
        };
      });

      toast.success('Milestone created successfully!');
      return { success: true, data: milestone };
    } catch (error) {
      const errorInfo = handleAPIError(error);
      toast.error(`Failed to create milestone: ${errorInfo.message}`);
      return { success: false, error: errorInfo.message };
    }
  };

  // Update milestone
  const updateMilestone = async (milestoneId, milestoneData) => {
    try {
      const response = await roadmapAPI.updateMilestone(milestoneId, milestoneData);

      // Reload roadmap data to get the updated state
      // await loadRoadmap();

      toast.success('Milestone updated successfully!');
      return { success: true, data: response };
    } catch (error) {
      const errorInfo = handleAPIError(error);
      toast.error(`Failed to update milestone: ${errorInfo.message}`);
      return { success: false, error: errorInfo.message };
    }
  };

  // Create epic
  const createEpic = async (epicData) => {
    try {
      const epic = await roadmapAPI.createEpic(epicData);

      // Optimistically update the roadmap data
      setRoadmapData(prevData => {
        if (!prevData) return prevData;

        const updatedPillars = prevData.pillars.map(pillar => ({
          ...pillar,
          milestones: pillar.milestones.map(milestone => {
            if (milestone.id === epicData.milestone_id) {
              return {
                ...milestone,
                epics: [...milestone.epics, epic],
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

      toast.success('Epic created successfully!');
      return { success: true, data: epic };
    } catch (error) {
      const errorInfo = handleAPIError(error);
      toast.error(`Failed to create epic: ${errorInfo.message}`);
      return { success: false, error: errorInfo.message };
    }
  };

  // Update epic
  const updateEpic = async (epicId, epicData) => {
    try {
      const response = await roadmapAPI.updateEpic(epicId, epicData);

      // Reload roadmap data to get the updated state
      // await loadRoadmap();

      toast.success('Epic updated successfully!');
      return { success: true, data: response };
    } catch (error) {
      const errorInfo = handleAPIError(error);
      toast.error(`Failed to update epic: ${errorInfo.message}`);
      return { success: false, error: errorInfo.message };
    }
  };

  // Move epic to different milestone
  const moveEpic = async (epicId, newMilestoneId) => {
    try {
      await roadmapAPI.updateEpicMilestone(epicId, newMilestoneId);

      // Optimistically update the roadmap data
      setRoadmapData(prevData => {
        if (!prevData) return prevData;

        let movedEpic = null;

        // Remove epic from current milestone
        const updatedPillars = prevData.pillars.map(pillar => ({
          ...pillar,
          milestones: pillar.milestones.map(milestone => ({
            ...milestone,
            epics: milestone.epics.filter(epic => {
              if (epic.id === epicId) {
                movedEpic = { ...epic, milestone_id: newMilestoneId };
                return false;
              }
              return true;
            }),
          })),
        }));

        // Add epic to new milestone
        const finalPillars = updatedPillars.map(pillar => ({
          ...pillar,
          milestones: pillar.milestones.map(milestone => {
            if (milestone.id === newMilestoneId && movedEpic) {
              return {
                ...milestone,
                epics: [...milestone.epics, movedEpic],
              };
            }
            return milestone;
          }),
        }));

        return {
          ...prevData,
          pillars: finalPillars,
        };
      });

      toast.success('Epic moved successfully!');
      return { success: true };
    } catch (error) {
      const errorInfo = handleAPIError(error);
      toast.error(`Failed to move epic: ${errorInfo.message}`);
      // Reload roadmap to ensure consistency
      // loadRoadmap();
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
